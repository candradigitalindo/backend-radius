package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrODPNotFound      = errors.New("ODP tidak ditemukan")
	ErrODPPortNotFound  = errors.New("Port ODP tidak ditemukan")
	ErrSplitterNotFound = errors.New("Splitter tidak ditemukan")
	// ErrTopology menandai pelanggaran aturan topologi (kapasitas ODC penuh,
	// line ganda, siklus induk, format rasio salah) — dipetakan handler ke 400.
	ErrTopology = errors.New("topologi tidak valid")
)

// parseSplitRatio membaca kapasitas keluaran dari tipe splitter "1:N".
// Mengembalikan 0 bila format tidak dikenali (tanpa batas kapasitas).
func parseSplitRatio(splitterType string) int {
	parts := strings.SplitN(strings.TrimSpace(splitterType), ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != "1" {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || n < 1 {
		return 0
	}
	return n
}

type ODPService struct {
	odpRepo      repository.ODPRepository
	customerRepo repository.CustomerRepository
	ontRepo      repository.ONTRepository
}

func NewODPService(odpRepo repository.ODPRepository, customerRepo repository.CustomerRepository, ontRepo repository.ONTRepository) *ODPService {
	return &ODPService{odpRepo: odpRepo, customerRepo: customerRepo, ontRepo: ontRepo}
}

// ODP operations

type CreateODPInput struct {
	TenantID      string
	OLTID         *string
	SplitterID    *string
	PONPortID     *string
	SplitterRatio *string
	Name          string
	Address       *string
	Latitude      float64
	Longitude     float64
	TotalPorts    int
	Sequence      int
	CableLengthM  float64
	RatioPercent  float64
	SplitterType  *string
	SplitterLine  *int
	Status        string
	Notes         *string
}

type UpdateODPInput struct {
	OLTID         *string
	SplitterID    *string
	PONPortID     *string
	SplitterRatio *string
	Name          string
	Address       *string
	Latitude      float64
	Longitude     float64
	TotalPorts    int
	Sequence      int
	CableLengthM  float64
	RatioPercent  float64
	SplitterType  *string
	SplitterLine  *int
	Status        string
	Notes         *string
}

func (s *ODPService) Create(ctx context.Context, input CreateODPInput) (*model.ODP, error) {
	odp := &model.ODP{
		TenantID:      input.TenantID,
		OLTID:         input.OLTID,
		SplitterID:    input.SplitterID,
		PONPortID:     input.PONPortID,
		SplitterRatio: input.SplitterRatio,
		Name:          input.Name,
		Address:       input.Address,
		Latitude:      input.Latitude,
		Longitude:     input.Longitude,
		TotalPorts:    input.TotalPorts,
		Sequence:      input.Sequence,
		CableLengthM:  input.CableLengthM,
		RatioPercent:  input.RatioPercent,
		SplitterType:  input.SplitterType,
		SplitterLine:  input.SplitterLine,
		Status:        input.Status,
		Notes:         input.Notes,
	}
	if odp.TotalPorts <= 0 {
		odp.TotalPorts = 8
	}
	if odp.Status == "" {
		odp.Status = "draft"
	}

	if err := s.validateODPParent(ctx, odp); err != nil {
		return nil, err
	}

	s.computePowerLevel(ctx, odp)

	if err := s.odpRepo.Create(ctx, odp); err != nil {
		return nil, err
	}

	// Materialisasi baris port 1..TotalPorts. Tanpa ini dropdown port di form
	// pelanggan kosong dan pelanggan FTTH tidak bisa ditautkan ke ODP.
	s.ensurePorts(ctx, odp)

	return s.odpRepo.FindByID(ctx, odp.TenantID, odp.ID)
}

// ensurePorts membuat baris odp_ports yang belum ada sampai nomor TotalPorts.
// Port yang sudah ada (termasuk yang sedang dipakai pelanggan) tidak disentuh.
func (s *ODPService) ensurePorts(ctx context.Context, odp *model.ODP) {
	if odp == nil || odp.TotalPorts <= 0 {
		return
	}
	existing, err := s.odpRepo.ListPorts(ctx, odp.ID)
	if err != nil {
		log.Printf("[ODPService] gagal membaca port ODP %s: %v", odp.ID, err)
		return
	}
	have := make(map[int]bool, len(existing))
	for _, p := range existing {
		have[p.PortNumber] = true
	}
	for n := 1; n <= odp.TotalPorts; n++ {
		if have[n] {
			continue
		}
		port := &model.ODPPort{ODPID: odp.ID, PortNumber: n, Status: "available"}
		if err := s.odpRepo.CreatePort(ctx, port); err != nil {
			log.Printf("[ODPService] gagal membuat port %d ODP %s: %v", n, odp.ID, err)
		}
	}
}

func (s *ODPService) GetByID(ctx context.Context, tenantID, odpID string) (*model.ODP, error) {
	odp, err := s.odpRepo.FindByID(ctx, tenantID, odpID)
	if err != nil {
		return nil, err
	}
	if odp == nil {
		return nil, ErrODPNotFound
	}
	populateSignalMetrics(odp)
	s.populateSplitterChain(ctx, odp)
	return odp, nil
}

// splitterLossDB: redaman splitter pasif per rasio (dB) — selaras dengan tabel
// SPLITTER_LOSS di frontend (useLinkBudget.ts).
var splitterLossDB = map[string]float64{
	"1:2": 3.70, "1:4": 7.40, "1:8": 10.38, "1:16": 13.50, "1:32": 17.10, "1:64": 20.50,
}

func splitterLoss(splitterType string) float64 {
	if loss, ok := splitterLossDB[strings.ReplaceAll(splitterType, " ", "")]; ok {
		return loss
	}
	if n := parseSplitRatio(splitterType); n > 1 {
		return 10 * math.Log10(float64(n)) // pembagian ideal tanpa excess loss
	}
	return 0
}

// populateSplitterChain melengkapi ODP ber-induk ODC dengan rantai ODC sampai
// root plus info PON port/OLT di ujungnya — untuk tampilan jalur & link budget.
func (s *ODPService) populateSplitterChain(ctx context.Context, odp *model.ODP) {
	if odp.SplitterID == nil || *odp.SplitterID == "" {
		return
	}
	chain, err := s.odpRepo.GetSplitterChain(ctx, odp.TenantID, *odp.SplitterID)
	if err != nil || len(chain) == 0 {
		return
	}
	odp.SplitterChain = chain

	root := chain[len(chain)-1]
	if root.PONPortID == nil || *root.PONPortID == "" {
		return
	}
	pp, olt, err := s.odpRepo.GetPONPortRoot(ctx, *root.PONPortID)
	if err != nil || pp == nil {
		return
	}
	odp.RootPONPortNumber = &pp.PortNumber
	odp.RootSFPRxPower = pp.SFPRxPower
	if olt != nil {
		odp.RootOLTName = &olt.Name
	}

	// Rantai ODP se-line + power efektif masuk ODP ini setelah tap pendahulu.
	if odp.SplitterLine == nil || pp.SFPRxPower == nil {
		return
	}
	lineChain, err := s.odpRepo.ListBySplitterLine(ctx, odp.TenantID, *odp.SplitterID, *odp.SplitterLine)
	if err != nil {
		return
	}
	odp.LineChain = lineChain

	start := *pp.SFPRxPower
	for _, sp := range chain {
		start -= splitterLoss(sp.SplitterType)
	}
	input := lineInputPower(start, lineChain, odp.ID, odp.Sequence)
	odp.LineInputPower = &input
}

// lineInputPower menghitung power yang masuk ke ODP target setelah melewati
// tap ODP-ODP pendahulunya dalam satu line (urut sequence): tiap pendahulu
// mengurangi rugi kabel segmennya lalu meneruskan sisa (pass-through) sesuai
// rasio tap-nya. Matematika sama dengan computePowerLevel mode estafet.
func lineInputPower(startDBm float64, lineChain []model.ODP, targetID string, targetSeq int) float64 {
	const fiberLossPerKm = 0.35
	remaining := startDBm
	for _, prev := range lineChain {
		if prev.ID == targetID || prev.Sequence >= targetSeq {
			continue
		}
		remaining -= (prev.CableLengthM / 1000.0) * fiberLossPerKm
		ratio := prev.RatioPercent
		if ratio <= 0 || ratio > 100 {
			ratio = 100
		}
		if ratio < 100 {
			remaining += 10 * math.Log10((100-ratio)/100.0)
		} else {
			remaining -= 30 // seluruh power diambil, praktis habis
		}
	}
	return math.Round(remaining*100) / 100
}

// validateODPParent menegakkan aturan topologi ODP ber-induk ODC: ODC harus
// milik tenant, kapasitas rasio tidak terlampaui, dan line keluaran valid &
// belum dipakai ODP lain. ODP tanpa ODC tidak menyimpan nomor line.
func (s *ODPService) validateODPParent(ctx context.Context, odp *model.ODP) error {
	if odp.SplitterID == nil || *odp.SplitterID == "" {
		odp.SplitterLine = nil
		return nil
	}

	splitter, err := s.odpRepo.FindSplitterByID(ctx, odp.TenantID, *odp.SplitterID)
	if err != nil {
		return err
	}
	if splitter == nil {
		return ErrSplitterNotFound
	}

	capacity := parseSplitRatio(splitter.SplitterType)

	// Satu line boleh berisi rantai beberapa ODP (estafet). Yang memakan
	// keluaran ODC adalah LINE-nya, bukan tiap ODP — jadi ODP baru di line
	// yang sudah berpenghuni tidak menambah pemakaian keluaran.
	newOutput := 1
	if odp.SplitterLine != nil {
		line := *odp.SplitterLine
		if line < 1 {
			return fmt.Errorf("%w: nomor line ODC minimal 1", ErrTopology)
		}
		if capacity > 0 && line > capacity {
			return fmt.Errorf("%w: line %d melebihi kapasitas ODC %s (%s)",
				ErrTopology, line, splitter.Name, splitter.SplitterType)
		}
		if odp.Sequence < 1 {
			odp.Sequence = 1
		}
		taken, err := s.odpRepo.FindODPBySplitterLineSeq(ctx, odp.TenantID, splitter.ID, line, odp.Sequence, odp.ID)
		if err != nil {
			return err
		}
		if taken != nil {
			return fmt.Errorf("%w: urutan %d pada line %d ODC %s sudah dipakai ODP %s — pakai nomor urutan berikutnya",
				ErrTopology, odp.Sequence, line, splitter.Name, taken.Name)
		}
		occupied, err := s.odpRepo.LineOccupied(ctx, odp.TenantID, splitter.ID, line, odp.ID)
		if err != nil {
			return err
		}
		if occupied {
			newOutput = 0 // menyambung rantai line yang sudah ada
		}
	}

	outputsUsed, err := s.odpRepo.CountSplitterOutputsUsed(ctx, odp.TenantID, splitter.ID, odp.ID)
	if err != nil {
		return err
	}
	subSplitters, err := s.odpRepo.CountChildSplitters(ctx, odp.TenantID, splitter.ID, "")
	if err != nil {
		return err
	}
	if capacity > 0 && outputsUsed+subSplitters+newOutput > capacity {
		return fmt.Errorf("%w: ODC %s (%s) sudah penuh — %d dari %d keluaran terpakai",
			ErrTopology, splitter.Name, splitter.SplitterType, outputsUsed+subSplitters, capacity)
	}
	return nil
}

// validateSplitterParent menegakkan aturan topologi ODC: format tipe 1:N,
// induk milik tenant, tidak membentuk siklus, dan kapasitas induk cukup.
// selfID kosong saat create (belum punya ID).
func (s *ODPService) validateSplitterParent(ctx context.Context, tenantID string, splitterType string, parentID *string, selfID string) error {
	if parseSplitRatio(splitterType) == 0 {
		return fmt.Errorf("%w: tipe splitter harus berformat 1:N (contoh 1:4)", ErrTopology)
	}
	if parentID == nil || *parentID == "" {
		return nil
	}
	if selfID != "" && *parentID == selfID {
		return fmt.Errorf("%w: ODC tidak boleh menginduk ke dirinya sendiri", ErrTopology)
	}

	parent, err := s.odpRepo.FindSplitterByID(ctx, tenantID, *parentID)
	if err != nil {
		return err
	}
	if parent == nil {
		return fmt.Errorf("%w: ODC induk tidak ditemukan", ErrTopology)
	}

	// Deteksi siklus: telusuri rantai induk; bila kembali ke diri sendiri, tolak.
	cur := parent
	for depth := 0; depth < 32 && cur != nil; depth++ {
		if selfID != "" && cur.ID == selfID {
			return fmt.Errorf("%w: induk yang dipilih membentuk siklus", ErrTopology)
		}
		if cur.ParentSplitterID == nil || *cur.ParentSplitterID == "" {
			break
		}
		next, err := s.odpRepo.FindSplitterByID(ctx, tenantID, *cur.ParentSplitterID)
		if err != nil {
			return err
		}
		cur = next
	}

	capacity := parseSplitRatio(parent.SplitterType)
	if capacity > 0 {
		outputsUsed, err := s.odpRepo.CountSplitterOutputsUsed(ctx, tenantID, parent.ID, "")
		if err != nil {
			return err
		}
		subSplitters, err := s.odpRepo.CountChildSplitters(ctx, tenantID, parent.ID, selfID)
		if err != nil {
			return err
		}
		if outputsUsed+subSplitters+1 > capacity {
			return fmt.Errorf("%w: ODC induk %s (%s) sudah penuh — %d dari %d keluaran terpakai",
				ErrTopology, parent.Name, parent.SplitterType, outputsUsed+subSplitters, capacity)
		}
	}
	return nil
}

func (s *ODPService) Update(ctx context.Context, tenantID, odpID string, input UpdateODPInput) (*model.ODP, error) {
	odp, err := s.odpRepo.FindByID(ctx, tenantID, odpID)
	if err != nil {
		return nil, err
	}
	if odp == nil {
		return nil, ErrODPNotFound
	}

	odp.OLTID = input.OLTID
	odp.SplitterID = input.SplitterID
	odp.PONPortID = input.PONPortID
	odp.SplitterRatio = input.SplitterRatio
	odp.Name = input.Name
	odp.Address = input.Address
	odp.Latitude = input.Latitude
	odp.Longitude = input.Longitude
	odp.TotalPorts = input.TotalPorts
	odp.Sequence = input.Sequence
	odp.CableLengthM = input.CableLengthM
	odp.RatioPercent = input.RatioPercent
	odp.SplitterType = input.SplitterType
	odp.SplitterLine = input.SplitterLine
	odp.Status = input.Status
	odp.Notes = input.Notes
	if odp.Status == "" {
		odp.Status = "draft"
	}

	if err := s.validateODPParent(ctx, odp); err != nil {
		return nil, err
	}

	s.computePowerLevel(ctx, odp)

	if err := s.odpRepo.Update(ctx, odp); err != nil {
		return nil, err
	}

	// Tambah baris port bila TotalPorts dinaikkan.
	s.ensurePorts(ctx, odp)

	return s.odpRepo.FindByID(ctx, tenantID, odpID)
}

func (s *ODPService) Delete(ctx context.Context, tenantID, odpID string) error {
	odp, err := s.odpRepo.FindByID(ctx, tenantID, odpID)
	if err != nil {
		return err
	}
	if odp == nil {
		return ErrODPNotFound
	}

	// Lepas rujukan pelanggan ke semua port ODP ini — FK customers.odp_port_id
	// bertipe NO ACTION, tanpa ini cascade penghapusan port ikut gagal.
	ports, err := s.odpRepo.ListPorts(ctx, odpID)
	if err != nil {
		return err
	}
	for _, p := range ports {
		if err := s.customerRepo.ClearODPPortID(ctx, p.ID); err != nil {
			return err
		}
	}

	return s.odpRepo.Delete(ctx, tenantID, odpID)
}

func (s *ODPService) List(ctx context.Context, tenantID string, filter repository.ODPFilter) ([]model.ODP, int, error) {
	odps, total, err := s.odpRepo.List(ctx, tenantID, filter)
	if err != nil {
		return nil, 0, err
	}
	for i := range odps {
		populateSignalMetrics(&odps[i])
	}
	return odps, total, nil
}

func populateSignalMetrics(odp *model.ODP) {
	if odp == nil || odp.PONPortSFPRxPower == nil || odp.PowerLevelDBm == nil {
		return
	}
	attenuation := math.Round((*odp.PONPortSFPRxPower-*odp.PowerLevelDBm)*100) / 100
	odp.SignalAttenuationDB = &attenuation
}

// computePowerLevel calculates the optical power at this ODP position.
// Power chain: start from PON port sfp_rx_power, then for each preceding ODP
// (ordered by sequence) subtract cable attenuation and splitter loss.
// Cable attenuation: 0.35 dB/km. Splitter loss: -10*log10(ratio_percent/100).
func (s *ODPService) computePowerLevel(ctx context.Context, odp *model.ODP) {
	if odp.PONPortID == nil || *odp.PONPortID == "" {
		// Mode via-ODC: power dihitung dari root PON port rantai ODC dikurangi
		// loss tiap splitter, lalu berantai per line seperti mode estafet.
		s.computePowerLevelViaODC(ctx, odp)
		return
	}

	// Get starting power from PON port SFP
	startPower, err := s.odpRepo.GetPONPortSFPRxPower(ctx, *odp.PONPortID)
	if err != nil || startPower == nil {
		odp.PowerLevelDBm = nil
		return
	}

	// Get all existing ODPs on the same PON port, ordered by sequence
	chain, _ := s.odpRepo.ListByPONPort(ctx, *odp.PONPortID)

	// Build the chain: include existing ODPs + replace/insert current ODP
	merged := make([]model.ODP, 0, len(chain)+1)
	replaced := false
	for _, c := range chain {
		if c.ID == odp.ID {
			merged = append(merged, *odp)
			replaced = true
		} else {
			merged = append(merged, c)
		}
	}
	if !replaced {
		merged = append(merged, *odp)
	}

	// Sort by sequence (simple insertion sort, chain is small)
	for i := 1; i < len(merged); i++ {
		for j := i; j > 0 && merged[j].Sequence < merged[j-1].Sequence; j-- {
			merged[j], merged[j-1] = merged[j-1], merged[j]
		}
	}

	// Walk the chain: remaining power starts at SFP rx power
	const fiberLossPerKm = 0.35
	remainingPowerDBm := *startPower

	for i := range merged {
		// Cable loss
		cableLossDB := (merged[i].CableLengthM / 1000.0) * fiberLossPerKm
		afterCable := remainingPowerDBm - cableLossDB

		// Power tapped by this ODP (what this ODP gets)
		ratio := merged[i].RatioPercent
		if ratio <= 0 || ratio > 100 {
			ratio = 100
		}
		tapDB := 10 * math.Log10(ratio/100.0)
		powerAtODP := afterCable + tapDB

		pRound := math.Round(powerAtODP*100) / 100
		merged[i].PowerLevelDBm = &pRound

		if merged[i].ID == odp.ID {
			odp.PowerLevelDBm = &pRound
		}

		// Remaining power for next ODP: pass-through fraction
		if ratio < 100 {
			passDB := 10 * math.Log10((100-ratio)/100.0)
			remainingPowerDBm = afterCable + passDB
		} else {
			// All power consumed, nothing left
			remainingPowerDBm = afterCable - 30 // effectively nothing
		}
	}
}

// computePowerLevelViaODC menghitung power_level_dbm untuk ODP ber-induk ODC:
// SFP root PON port − loss tiap splitter di rantai ODC − rugi/tap ODP
// pendahulu se-line − rugi kabel + tap ODP ini sendiri.
func (s *ODPService) computePowerLevelViaODC(ctx context.Context, odp *model.ODP) {
	odp.PowerLevelDBm = nil
	if odp.SplitterID == nil || *odp.SplitterID == "" {
		return
	}
	chain, err := s.odpRepo.GetSplitterChain(ctx, odp.TenantID, *odp.SplitterID)
	if err != nil || len(chain) == 0 {
		return
	}
	root := chain[len(chain)-1]
	if root.PONPortID == nil || *root.PONPortID == "" {
		return
	}
	startPower, err := s.odpRepo.GetPONPortSFPRxPower(ctx, *root.PONPortID)
	if err != nil || startPower == nil {
		return
	}

	remaining := *startPower
	for _, sp := range chain {
		remaining -= splitterLoss(sp.SplitterType)
	}

	if odp.SplitterLine != nil {
		lineChain, err := s.odpRepo.ListBySplitterLine(ctx, odp.TenantID, *odp.SplitterID, *odp.SplitterLine)
		if err == nil {
			remaining = lineInputPower(remaining, lineChain, odp.ID, odp.Sequence)
		}
	}

	const fiberLossPerKm = 0.35
	remaining -= (odp.CableLengthM / 1000.0) * fiberLossPerKm
	ratio := odp.RatioPercent
	if ratio <= 0 || ratio > 100 {
		ratio = 100
	}
	power := remaining + 10*math.Log10(ratio/100.0)
	pRound := math.Round(power*100) / 100
	odp.PowerLevelDBm = &pRound
}

// ODP Port operations

type CreateODPPortInput struct {
	ODPID      string
	PortNumber int
	CustomerID *string
	Notes      *string
}

type UpdateODPPortInput struct {
	CustomerID *string
	Status     string
	Notes      *string
}

func (s *ODPService) CreatePort(ctx context.Context, tenantID string, input CreateODPPortInput) (*model.ODPPort, error) {
	odp, err := s.odpRepo.FindByID(ctx, tenantID, input.ODPID)
	if err != nil {
		return nil, err
	}
	if odp == nil {
		return nil, ErrODPNotFound
	}

	port := &model.ODPPort{
		ODPID:      input.ODPID,
		PortNumber: input.PortNumber,
		CustomerID: input.CustomerID,
		Status:     "available",
		Notes:      input.Notes,
	}
	if input.CustomerID != nil {
		port.Status = "used"
	}

	if err := s.odpRepo.CreatePort(ctx, port); err != nil {
		return nil, err
	}
	return s.odpRepo.FindPortByID(ctx, port.ID)
}

func (s *ODPService) ListPorts(ctx context.Context, tenantID, odpID string) ([]model.ODPPort, error) {
	odp, err := s.odpRepo.FindByID(ctx, tenantID, odpID)
	if err != nil {
		return nil, err
	}
	if odp == nil {
		return nil, ErrODPNotFound
	}
	return s.odpRepo.ListPorts(ctx, odpID)
}

func (s *ODPService) UpdatePort(ctx context.Context, tenantID, portID string, input UpdateODPPortInput) (*model.ODPPort, error) {
	port, err := s.odpRepo.FindPortByID(ctx, portID)
	if err != nil {
		return nil, err
	}
	if port == nil {
		return nil, ErrODPPortNotFound
	}

	// Verify the port's ODP belongs to this tenant
	odp, err := s.odpRepo.FindByID(ctx, tenantID, port.ODPID)
	if err != nil {
		return nil, err
	}
	if odp == nil {
		return nil, ErrODPPortNotFound
	}

	oldCustomerID := port.CustomerID
	newCustomerID := input.CustomerID

	port.CustomerID = newCustomerID
	port.Status = input.Status
	port.Notes = input.Notes

	// Auto-set status based on customer assignment
	if newCustomerID != nil && *newCustomerID != "" {
		port.Status = "used"
	} else if newCustomerID == nil || *newCustomerID == "" {
		port.CustomerID = nil
		if port.Status == "used" {
			port.Status = "available"
		}
	}

	if err := s.odpRepo.UpdatePort(ctx, port); err != nil {
		return nil, err
	}

	// Sync: clear old customer's odp_port_id
	oldCustIDStr := ""
	if oldCustomerID != nil {
		oldCustIDStr = *oldCustomerID
	}
	newCustIDStr := ""
	if newCustomerID != nil {
		newCustIDStr = *newCustomerID
	}

	customerChanged := oldCustIDStr != newCustIDStr

	if customerChanged {
		// Clear old customer's odp_port_id
		if oldCustIDStr != "" {
			_ = s.customerRepo.UpdateODPPortID(ctx, oldCustIDStr, nil)
		}
		// Set new customer's odp_port_id
		if newCustIDStr != "" {
			_ = s.customerRepo.UpdateODPPortID(ctx, newCustIDStr, &portID)
		}
		// Sync ONT: update customer_id on the ONT linked to this port
		_ = s.ontRepo.UpdateCustomerByODPPortID(ctx, portID, port.CustomerID)
	}

	return s.odpRepo.FindPortByID(ctx, port.ID)
}

func (s *ODPService) DeletePort(ctx context.Context, tenantID, portID string) error {
	port, err := s.odpRepo.FindPortByID(ctx, portID)
	if err != nil {
		return err
	}
	if port == nil {
		return ErrODPPortNotFound
	}

	// Verify the port's ODP belongs to this tenant
	odp, err := s.odpRepo.FindByID(ctx, tenantID, port.ODPID)
	if err != nil {
		return err
	}
	if odp == nil {
		return ErrODPPortNotFound
	}

	// Lepas rujukan pelanggan dulu — FK customers.odp_port_id bertipe NO ACTION,
	// tanpa ini penghapusan port yang masih ditunjuk pelanggan gagal.
	if err := s.customerRepo.ClearODPPortID(ctx, portID); err != nil {
		return err
	}

	return s.odpRepo.DeletePort(ctx, portID)
}

// Splitter operations

type CreateSplitterInput struct {
	TenantID         string
	PONPortID        *string
	ParentSplitterID *string
	Name             string
	SplitterType     string
	Latitude         *float64
	Longitude        *float64
	Notes            *string
}

type UpdateSplitterInput struct {
	PONPortID        *string
	ParentSplitterID *string
	Name             string
	SplitterType     string
	Latitude         *float64
	Longitude        *float64
	Notes            *string
}

func (s *ODPService) CreateSplitter(ctx context.Context, input CreateSplitterInput) (*model.Splitter, error) {
	if err := s.validateSplitterParent(ctx, input.TenantID, input.SplitterType, input.ParentSplitterID, ""); err != nil {
		return nil, err
	}

	splitter := &model.Splitter{
		TenantID:         input.TenantID,
		PONPortID:        input.PONPortID,
		ParentSplitterID: input.ParentSplitterID,
		Name:             input.Name,
		SplitterType:     input.SplitterType,
		Latitude:         input.Latitude,
		Longitude:        input.Longitude,
		Notes:            input.Notes,
	}

	if err := s.odpRepo.CreateSplitter(ctx, splitter); err != nil {
		return nil, err
	}
	return s.odpRepo.FindSplitterByID(ctx, splitter.TenantID, splitter.ID)
}

func (s *ODPService) GetSplitterByID(ctx context.Context, tenantID, splitterID string) (*model.Splitter, error) {
	splitter, err := s.odpRepo.FindSplitterByID(ctx, tenantID, splitterID)
	if err != nil {
		return nil, err
	}
	if splitter == nil {
		return nil, ErrSplitterNotFound
	}
	return splitter, nil
}

func (s *ODPService) UpdateSplitter(ctx context.Context, tenantID, splitterID string, input UpdateSplitterInput) (*model.Splitter, error) {
	splitter, err := s.odpRepo.FindSplitterByID(ctx, tenantID, splitterID)
	if err != nil {
		return nil, err
	}
	if splitter == nil {
		return nil, ErrSplitterNotFound
	}

	if err := s.validateSplitterParent(ctx, tenantID, input.SplitterType, input.ParentSplitterID, splitterID); err != nil {
		return nil, err
	}

	splitter.PONPortID = input.PONPortID
	splitter.ParentSplitterID = input.ParentSplitterID
	splitter.Name = input.Name
	splitter.SplitterType = input.SplitterType
	splitter.Latitude = input.Latitude
	splitter.Longitude = input.Longitude
	splitter.Notes = input.Notes

	if err := s.odpRepo.UpdateSplitter(ctx, splitter); err != nil {
		return nil, err
	}
	return s.odpRepo.FindSplitterByID(ctx, tenantID, splitterID)
}

func (s *ODPService) DeleteSplitter(ctx context.Context, tenantID, splitterID string) error {
	splitter, err := s.odpRepo.FindSplitterByID(ctx, tenantID, splitterID)
	if err != nil {
		return err
	}
	if splitter == nil {
		return ErrSplitterNotFound
	}
	return s.odpRepo.DeleteSplitter(ctx, tenantID, splitterID)
}

func (s *ODPService) ListSplitters(ctx context.Context, tenantID string, filter repository.SplitterFilter) ([]model.Splitter, int, error) {
	return s.odpRepo.ListSplitters(ctx, tenantID, filter)
}
