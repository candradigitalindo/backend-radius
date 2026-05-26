package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os/exec"
	"strings"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/vpn"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrOLTNotFound     = errors.New("OLT tidak ditemukan")
	ErrPONPortNotFound = errors.New("Port PON tidak ditemukan")
)

type OLTService struct {
	oltRepo    repository.OLTRepository
	vpnManager *vpn.Manager
}

func NewOLTService(oltRepo repository.OLTRepository) *OLTService {
	return &OLTService{oltRepo: oltRepo}
}

func (s *OLTService) WithVPNManager(vpnMgr *vpn.Manager) *OLTService {
	s.vpnManager = vpnMgr
	return s
}

// provisionRoute adds WireGuard AllowedIP and OS route for the OLT subnet.
// Called automatically on Create and Update so routing is ready immediately.
func (s *OLTService) provisionRoute(olt *model.OLT) {
	if olt.Router == nil || olt.Router.VPNIP == "" {
		return
	}
	ip := net.ParseIP(olt.IPAddress)
	if ip == nil {
		return
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return
	}
	subnet := fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
	gateway := olt.Router.VPNIP

	if s.vpnManager != nil {
		if err := s.vpnManager.AddPeerAllowedIP(gateway, subnet); err != nil {
			log.Printf("[OLT] Failed to add WireGuard AllowedIP %s to peer %s: %v", subnet, gateway, err)
		} else {
			log.Printf("[OLT] Added WireGuard AllowedIP %s to peer %s", subnet, gateway)
		}
	}

	out, _ := exec.Command("ip", "route", "show", subnet).Output()
	if strings.Contains(string(out), gateway) {
		return // route already exists
	}
	if err := exec.Command("ip", "route", "add", subnet, "via", gateway, "dev", "wg0").Run(); err != nil {
		log.Printf("[OLT] Failed to add route %s via %s: %v", subnet, gateway, err)
		return
	}
	log.Printf("[OLT] Auto-added route %s via %s (OLT: %s)", subnet, gateway, olt.Name)
}

type CreateOLTInput struct {
	TenantID      string
	RouterID      *string
	Name          string
	IPAddress     string
	Vendor        *string
	Model         *string
	SerialNumber  *string
	TotalPONPorts int
	Latitude      *float64
	Longitude     *float64
	SNMPCommunity string
	Notes         *string
}

type UpdateOLTInput struct {
	RouterID      *string
	Name          string
	IPAddress     string
	Vendor        *string
	Model         *string
	SerialNumber  *string
	TotalPONPorts int
	Latitude      *float64
	Longitude     *float64
	SNMPCommunity string
	Status        string
	Notes         *string
}

func (s *OLTService) Create(ctx context.Context, input CreateOLTInput) (*model.OLT, error) {
	olt := &model.OLT{
		TenantID:      input.TenantID,
		RouterID:      input.RouterID,
		Name:          input.Name,
		IPAddress:     input.IPAddress,
		Vendor:        input.Vendor,
		Model:         input.Model,
		SerialNumber:  input.SerialNumber,
		TotalPONPorts: input.TotalPONPorts,
		Latitude:      input.Latitude,
		Longitude:     input.Longitude,
		SNMPCommunity: input.SNMPCommunity,
		Status:        "active",
		Notes:         input.Notes,
	}
	if olt.SNMPCommunity == "" {
		olt.SNMPCommunity = "public"
	}
	if olt.TotalPONPorts <= 0 {
		olt.TotalPONPorts = 16
	}

	if err := s.oltRepo.Create(ctx, olt); err != nil {
		return nil, err
	}
	created, err := s.oltRepo.FindByID(ctx, olt.TenantID, olt.ID)
	if err != nil {
		return nil, err
	}
	go s.provisionRoute(created)
	return created, nil
}

func (s *OLTService) GetByID(ctx context.Context, tenantID, oltID string) (*model.OLT, error) {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}
	return olt, nil
}

func (s *OLTService) Update(ctx context.Context, tenantID, oltID string, input UpdateOLTInput) (*model.OLT, error) {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}

	olt.RouterID = input.RouterID
	olt.Name = input.Name
	olt.IPAddress = input.IPAddress
	olt.Vendor = input.Vendor
	olt.Model = input.Model
	olt.SerialNumber = input.SerialNumber
	olt.TotalPONPorts = input.TotalPONPorts
	olt.Latitude = input.Latitude
	olt.Longitude = input.Longitude
	olt.SNMPCommunity = input.SNMPCommunity
	olt.Status = input.Status
	olt.Notes = input.Notes

	if err := s.oltRepo.Update(ctx, olt); err != nil {
		return nil, err
	}
	updated, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	go s.provisionRoute(updated)
	return updated, nil
}

func (s *OLTService) Delete(ctx context.Context, tenantID, oltID string) error {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return err
	}
	if olt == nil {
		return ErrOLTNotFound
	}
	return s.oltRepo.Delete(ctx, tenantID, oltID)
}

func (s *OLTService) List(ctx context.Context, tenantID string, filter repository.OLTFilter) ([]model.OLT, int, error) {
	return s.oltRepo.List(ctx, tenantID, filter)
}

type CreatePONPortInput struct {
	OLTID       string
	PortNumber  int
	Description *string
	SFPRxPower  *float64
	SFPTxPower  *float64
}

type UpdatePONPortInput struct {
	Description *string
	Status      string
	SFPRxPower  *float64
	SFPTxPower  *float64
}

func (s *OLTService) CreatePONPort(ctx context.Context, tenantID string, input CreatePONPortInput) (*model.PONPort, error) {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, input.OLTID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}

	port := &model.PONPort{
		OLTID:       input.OLTID,
		PortNumber:  input.PortNumber,
		Description: input.Description,
		Status:      "active",
		SFPRxPower:  input.SFPRxPower,
		SFPTxPower:  input.SFPTxPower,
	}

	if err := s.oltRepo.CreatePONPort(ctx, port); err != nil {
		return nil, err
	}
	return s.oltRepo.FindPONPortByID(ctx, port.ID)
}

func (s *OLTService) ListPONPorts(ctx context.Context, tenantID, oltID string) ([]model.PONPort, error) {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}
	return s.oltRepo.ListPONPorts(ctx, oltID)
}

func (s *OLTService) UpdatePONPort(ctx context.Context, tenantID, oltID, portID string, input UpdatePONPortInput) (*model.PONPort, error) {
	// Verify OLT belongs to tenant
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}

	port, err := s.oltRepo.FindPONPortByID(ctx, portID)
	if err != nil {
		return nil, err
	}
	if port == nil || port.OLTID != oltID {
		return nil, ErrPONPortNotFound
	}

	port.Description = input.Description
	port.Status = input.Status
	port.SFPRxPower = input.SFPRxPower
	port.SFPTxPower = input.SFPTxPower

	if err := s.oltRepo.UpdatePONPort(ctx, port); err != nil {
		return nil, err
	}
	return s.oltRepo.FindPONPortByID(ctx, port.ID)
}

func (s *OLTService) DeletePONPort(ctx context.Context, tenantID, oltID, portID string) error {
	// Verify OLT belongs to tenant
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return err
	}
	if olt == nil {
		return ErrOLTNotFound
	}

	port, err := s.oltRepo.FindPONPortByID(ctx, portID)
	if err != nil {
		return err
	}
	if port == nil || port.OLTID != oltID {
		return ErrPONPortNotFound
	}
	return s.oltRepo.DeletePONPort(ctx, portID)
}
