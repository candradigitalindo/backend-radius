package service

import (
	"context"
	"fmt"
	"log"
	"net"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/vpn"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

type SNMPService struct {
	oltRepo    repository.OLTRepository
	vpnManager *vpn.Manager
	routeMu    sync.Mutex
	routeCache map[string]bool
}

func NewSNMPService(oltRepo repository.OLTRepository, vpnManager *vpn.Manager) *SNMPService {
	return &SNMPService{
		oltRepo:    oltRepo,
		vpnManager: vpnManager,
		routeCache: make(map[string]bool),
	}
}

func (s *SNMPService) ensureRoute(olt *model.OLT) {
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

	s.routeMu.Lock()
	defer s.routeMu.Unlock()

	if s.vpnManager != nil {
		if err := s.vpnManager.AddPeerAllowedIP(gateway, subnet); err != nil {
			log.Printf("[SNMP] Failed to add AllowedIP %s to peer %s: %v", subnet, gateway, err)
			return
		}
	}

	out, _ := exec.Command("ip", "route", "show", subnet).Output()
	if strings.Contains(string(out), gateway) {
		return
	}

	if err := exec.Command("ip", "route", "add", subnet, "via", gateway, "dev", "wg0").Run(); err != nil {
		log.Printf("[SNMP] Failed to add route %s via %s: %v", subnet, gateway, err)
		return
	}
	log.Printf("[SNMP] Auto-added route %s via %s (OLT: %s)", subnet, gateway, olt.Name)
}

const (
	oidSysDescr    = ".1.3.6.1.2.1.1.1.0"
	oidSysName     = ".1.3.6.1.2.1.1.5.0"
	oidSysUptime   = ".1.3.6.1.2.1.1.3.0"
	oidSysContact  = ".1.3.6.1.2.1.1.4.0"
	oidSysLocation = ".1.3.6.1.2.1.1.6.0"

	oidIfDescr       = ".1.3.6.1.2.1.2.2.1.2"
	oidIfType        = ".1.3.6.1.2.1.2.2.1.3"
	oidIfSpeed       = ".1.3.6.1.2.1.2.2.1.5"
	oidIfAdminStatus = ".1.3.6.1.2.1.2.2.1.7"
	oidIfOperStatus  = ".1.3.6.1.2.1.2.2.1.8"
	oidIfHCInOctets  = ".1.3.6.1.2.1.31.1.1.1.6"
	oidIfHCOutOctets = ".1.3.6.1.2.1.31.1.1.1.10"

	oidHrProcessorLoad = ".1.3.6.1.2.1.25.3.3.1.2"
	oidHrStorageDescr  = ".1.3.6.1.2.1.25.2.3.1.3"
	oidHrStorageAlloc  = ".1.3.6.1.2.1.25.2.3.1.4"
	oidHrStorageSize   = ".1.3.6.1.2.1.25.2.3.1.5"
	oidHrStorageUsed   = ".1.3.6.1.2.1.25.2.3.1.6"

	oidHiosoPonPortIndex = ".1.3.6.1.4.1.25355.3.1.8.3.1.1"
	oidHiosoOLTInfo      = ".1.3.6.1.4.1.25355.3.1.8.1.1.2"
	oidHiosoTemperature  = ".1.3.6.1.4.1.25355.3.1.8.1.1.9"
	oidHiosoUNITable = ".1.3.6.1.4.1.25355.3.2.6.1.2.1.1.1"
)

type OLTMonitorResult struct {
	Status     string             `json:"status"`
	Uptime     string             `json:"uptime"`
	UptimeSec  uint32             `json:"uptime_seconds"`
	System     MonitorSystem      `json:"system"`
	Resources  MonitorResources   `json:"resources"`
	Interfaces []MonitorInterface `json:"interfaces"`
	ONUSummary *MonitorONUSummary `json:"onu_summary,omitempty"`
	PolledAt   time.Time          `json:"polled_at"`
}

type MonitorSystem struct {
	Description string `json:"description"`
	Name        string `json:"name"`
	Contact     string `json:"contact,omitempty"`
	Location    string `json:"location,omitempty"`
	Firmware    string `json:"firmware,omitempty"`
}

type MonitorResources struct {
	CPUPercent    *int     `json:"cpu_percent,omitempty"`
	MemoryTotalKB *int64   `json:"memory_total_kb,omitempty"`
	MemoryUsedKB  *int64   `json:"memory_used_kb,omitempty"`
	MemoryPercent *float64 `json:"memory_percent,omitempty"`
	Temperature   *float64 `json:"temperature,omitempty"`
}

type MonitorInterface struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	AdminStatus string `json:"admin_status"`
	OperStatus  string `json:"oper_status"`
	SpeedMbps   int64  `json:"speed_mbps"`
	InOctets    uint64 `json:"in_octets"`
	OutOctets   uint64 `json:"out_octets"`
}

type MonitorONUSummary struct {
	Total  int          `json:"total"`
	Online int          `json:"online"`
	ONUs   []MonitorONU `json:"onus,omitempty"`
}

type MonitorONU struct {
	Index   int    `json:"index"`
	PONPort int    `json:"pon_port"`
	Serial  string `json:"serial"`
	Status  string `json:"status"`
}

func (s *SNMPService) newSNMPClient(target, community string) *gosnmp.GoSNMP {
	return &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   5 * time.Second,
		Retries:   1,
	}
}

func (s *SNMPService) MonitorOLT(ctx context.Context, tenantID, oltID string) (*OLTMonitorResult, error) {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil {
		return nil, err
	}
	if olt == nil {
		return nil, ErrOLTNotFound
	}

	s.ensureRoute(olt)

	community := olt.SNMPCommunity
	if community == "" {
		community = "public"
	}

	client := s.newSNMPClient(olt.IPAddress, community)
	if err := client.Connect(); err != nil {
		return &OLTMonitorResult{Status: "offline", PolledAt: time.Now()}, nil
	}
	defer client.Conn.Close()

	result := &OLTMonitorResult{Status: "online", PolledAt: time.Now(), Interfaces: []MonitorInterface{}}

	sysResp, err := client.Get([]string{
		oidSysDescr, oidSysName, oidSysUptime,
		oidSysContact, oidSysLocation,
	})
	if err != nil {
		return &OLTMonitorResult{Status: "offline", PolledAt: time.Now()}, nil
	}
	for _, v := range sysResp.Variables {
		switch v.Name {
		case oidSysDescr:
			result.System.Description = cleanSNMPString(snmpString(v))
		case oidSysName:
			result.System.Name = cleanSNMPString(snmpString(v))
		case oidSysUptime:
			if ticks, ok := v.Value.(uint32); ok {
				result.UptimeSec = ticks / 100
				result.Uptime = formatUptime(ticks)
			}
		case oidSysContact:
			result.System.Contact = cleanSNMPString(snmpString(v))
		case oidSysLocation:
			result.System.Location = cleanSNMPString(snmpString(v))
		}
	}

	ifaces, _ := s.pollInterfaces(client)
	result.Interfaces = ifaces

	s.pollCPU(client, &result.Resources)
	s.pollMemory(client, &result.Resources)

	vendor := ""
	if olt.Vendor != nil {
		vendor = strings.ToLower(*olt.Vendor)
	}
	isHioso := strings.Contains(vendor, "hioso")
	if !isHioso && vendor == "" {
		hiResult, _ := client.BulkWalkAll(oidHiosoPonPortIndex)
		isHioso = len(hiResult) > 0
	}

	if isHioso {
		s.pollHIOSOMonitor(client, result)
	} else {
		s.pollGenericVendorMonitor(client, vendor, result)
	}

	return result, nil
}

func (s *SNMPService) pollInterfaces(client *gosnmp.GoSNMP) ([]MonitorInterface, error) {
	descResults, err := client.BulkWalkAll(oidIfDescr)
	if err != nil {
		return []MonitorInterface{}, err
	}

	ifMap := make(map[int]*MonitorInterface)
	for _, v := range descResults {
		idx := extractIfIndex(v.Name, oidIfDescr)
		if idx <= 0 {
			continue
		}
		name := cleanSNMPString(snmpString(v))
		if name == "lo" {
			continue
		}
		ifMap[idx] = &MonitorInterface{Index: idx, Name: name}
	}
	if len(ifMap) == 0 {
		return []MonitorInterface{}, nil
	}

	typeResults, _ := client.BulkWalkAll(oidIfType)
	for _, v := range typeResults {
		idx := extractIfIndex(v.Name, oidIfType)
		if iface, ok := ifMap[idx]; ok {
			iface.Type = ifTypeString(v)
		}
	}

	adminResults, _ := client.BulkWalkAll(oidIfAdminStatus)
	for _, v := range adminResults {
		idx := extractIfIndex(v.Name, oidIfAdminStatus)
		if iface, ok := ifMap[idx]; ok {
			iface.AdminStatus = ifStatusString(v)
		}
	}

	operResults, _ := client.BulkWalkAll(oidIfOperStatus)
	for _, v := range operResults {
		idx := extractIfIndex(v.Name, oidIfOperStatus)
		if iface, ok := ifMap[idx]; ok {
			iface.OperStatus = ifStatusString(v)
		}
	}

	speedResults, _ := client.BulkWalkAll(oidIfSpeed)
	for _, v := range speedResults {
		idx := extractIfIndex(v.Name, oidIfSpeed)
		if iface, ok := ifMap[idx]; ok {
			if val, ok := v.Value.(uint); ok {
				iface.SpeedMbps = int64(val) / 1_000_000
			}
		}
	}

	hcInResults, _ := client.BulkWalkAll(oidIfHCInOctets)
	for _, v := range hcInResults {
		idx := extractIfIndex(v.Name, oidIfHCInOctets)
		if iface, ok := ifMap[idx]; ok {
			if val, ok := v.Value.(uint64); ok {
				iface.InOctets = val
			}
		}
	}

	hcOutResults, _ := client.BulkWalkAll(oidIfHCOutOctets)
	for _, v := range hcOutResults {
		idx := extractIfIndex(v.Name, oidIfHCOutOctets)
		if iface, ok := ifMap[idx]; ok {
			if val, ok := v.Value.(uint64); ok {
				iface.OutOctets = val
			}
		}
	}

	result := make([]MonitorInterface, 0, len(ifMap))
	indices := make([]int, 0, len(ifMap))
	for idx := range ifMap {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	for _, i := range indices {
		result = append(result, *ifMap[i])
	}
	return result, nil
}

func (s *SNMPService) pollCPU(client *gosnmp.GoSNMP, res *MonitorResources) {
	results, err := client.BulkWalkAll(oidHrProcessorLoad)
	if err != nil || len(results) == 0 {
		return
	}
	total, count := 0, 0
	for _, v := range results {
		if val, ok := v.Value.(int); ok {
			total += val
			count++
		}
	}
	if count > 0 {
		avg := total / count
		res.CPUPercent = &avg
	}
}

func (s *SNMPService) pollMemory(client *gosnmp.GoSNMP, res *MonitorResources) {
	descrResults, _ := client.BulkWalkAll(oidHrStorageDescr)
	allocResults, _ := client.BulkWalkAll(oidHrStorageAlloc)
	sizeResults, _ := client.BulkWalkAll(oidHrStorageSize)
	usedResults, _ := client.BulkWalkAll(oidHrStorageUsed)

	physIdx := -1
	for _, v := range descrResults {
		if strings.Contains(strings.ToLower(snmpString(v)), "physical memory") {
			physIdx = extractIfIndex(v.Name, oidHrStorageDescr)
			break
		}
	}
	if physIdx < 0 {
		return
	}

	var allocUnit int64 = 1024
	for _, v := range allocResults {
		if extractIfIndex(v.Name, oidHrStorageAlloc) == physIdx {
			if val, ok := v.Value.(int); ok {
				allocUnit = int64(val)
			}
		}
	}
	for _, v := range sizeResults {
		if extractIfIndex(v.Name, oidHrStorageSize) == physIdx {
			if val, ok := v.Value.(int); ok {
				total := int64(val) * allocUnit / 1024
				res.MemoryTotalKB = &total
			}
		}
	}
	for _, v := range usedResults {
		if extractIfIndex(v.Name, oidHrStorageUsed) == physIdx {
			if val, ok := v.Value.(int); ok {
				used := int64(val) * allocUnit / 1024
				res.MemoryUsedKB = &used
			}
		}
	}

	if res.MemoryTotalKB != nil && res.MemoryUsedKB != nil && *res.MemoryTotalKB > 0 {
		pct := float64(*res.MemoryUsedKB) / float64(*res.MemoryTotalKB) * 100
		res.MemoryPercent = &pct
	}
}

func (s *SNMPService) pollHIOSOMonitor(client *gosnmp.GoSNMP, result *OLTMonitorResult) {
	tempResults, _ := client.BulkWalkAll(oidHiosoTemperature)
	for _, v := range tempResults {
		raw := snmpFloat(v)
		if raw > 0 {
			temp := raw / 100.0
			result.Resources.Temperature = &temp
		}
	}

	infoResults, _ := client.BulkWalkAll(oidHiosoOLTInfo)
	if len(infoResults) > 0 {
		result.System.Firmware = cleanSNMPString(snmpString(infoResults[0]))
	}

	s.pollHIOSOONU(client, result)
}

func (s *SNMPService) pollHIOSOONU(client *gosnmp.GoSNMP, result *OLTMonitorResult) {
	uniResults, err := client.BulkWalkAll(oidHiosoUNITable)
	if err != nil || len(uniResults) == 0 {
		return
	}

	type onuKey struct{ port, onu int }
	onuSet := make(map[onuKey]struct{})

	for _, v := range uniResults {
		suffix := strings.TrimPrefix(v.Name, oidHiosoUNITable+".")
		if suffix == v.Name {
			continue
		}
		parts := strings.SplitN(suffix, ".", 3)
		if len(parts) < 2 {
			continue
		}
		var port, onu int
		fmt.Sscanf(parts[0], "%d", &port)
		fmt.Sscanf(parts[1], "%d", &onu)
		if port > 0 && onu > 0 {
			onuSet[onuKey{port, onu}] = struct{}{}
		}
	}

	summary := &MonitorONUSummary{Total: len(onuSet)}
	onuIndex := 1
	for key := range onuSet {
		summary.Online++
		summary.ONUs = append(summary.ONUs, MonitorONU{
			Index:   onuIndex,
			PONPort: key.port,
			Serial:  fmt.Sprintf("EPON-P%d-ONU%d", key.port, key.onu),
			Status:  "online",
		})
		onuIndex++
	}
	result.ONUSummary = summary
}

func (s *SNMPService) pollGenericVendorMonitor(client *gosnmp.GoSNMP, vendor string, result *OLTMonitorResult) {
	tempOID := ".1.3.6.1.2.1.99.1.1.1.4"
	typeOID := ".1.3.6.1.2.1.99.1.1.1.1"
	tempResults, _ := client.BulkWalkAll(tempOID)
	typeResults, _ := client.BulkWalkAll(typeOID)

	tempSensors := make(map[int]bool)
	for _, v := range typeResults {
		idx := extractIfIndex(v.Name, typeOID)
		if val, ok := v.Value.(int); ok && val == 8 {
			tempSensors[idx] = true
		}
	}
	for _, v := range tempResults {
		idx := extractIfIndex(v.Name, tempOID)
		if tempSensors[idx] {
			val := snmpFloat(v)
			if val > 0 {
				result.Resources.Temperature = &val
				break
			}
		}
	}
}

func snmpString(v gosnmp.SnmpPDU) string {
	switch v.Type {
	case gosnmp.OctetString:
		if v.Value == nil {
			return ""
		}
		if b, ok := v.Value.([]byte); ok {
			return string(b)
		}
		return ""
	default:
		if v.Value == nil {
			return ""
		}
		return fmt.Sprintf("%v", v.Value)
	}
}

func cleanSNMPString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	s = strings.TrimSpace(s)
	return s
}

func snmpFloat(v gosnmp.SnmpPDU) float64 {
	switch val := v.Value.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case uint:
		return float64(val)
	case int64:
		return float64(val)
	case uint64:
		return float64(val)
	case uint32:
		return float64(val)
	default:
		return 0
	}
}

func formatUptime(ticks uint32) string {
	totalSeconds := ticks / 100
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%d hari %d jam %d menit", days, hours, minutes)
	}
	return fmt.Sprintf("%d jam %d menit", hours, minutes)
}

func extractIfIndex(oid, base string) int {
	suffix := strings.TrimPrefix(oid, base+".")
	if suffix == oid {
		return 0
	}
	var idx int
	fmt.Sscanf(suffix, "%d", &idx)
	return idx
}

func ifStatusString(v gosnmp.SnmpPDU) string {
	val, ok := v.Value.(int)
	if !ok {
		return "unknown"
	}
	switch val {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	default:
		return "unknown"
	}
}

func ifTypeString(v gosnmp.SnmpPDU) string {
	val, ok := v.Value.(int)
	if !ok {
		return "other"
	}
	switch val {
	case 6:
		return "ethernet"
	case 24:
		return "loopback"
	case 53:
		return "propVirtual"
	case 117:
		return "gigabitEthernet"
	case 131:
		return "tunnel"
	case 135:
		return "l2vlan"
	case 161:
		return "ieee8023adLag"
	case 209:
		return "gpon"
	case 249:
		return "epon"
	case 250:
		return "tenGigabitEthernet"
	default:
		return fmt.Sprintf("type(%d)", val)
	}
}
