package service

import (
	"context"
	"errors"
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

// provisionRoute adds WireGuard AllowedIP and OS route for the OLT's host /32 route.
// Uses host route (/32) instead of /24 assumption — accurate regardless of OLT subnet size.
func (s *OLTService) provisionRoute(olt *model.OLT) {
	if olt.Router == nil || olt.Router.VPNIP == "" {
		return
	}
	ip := net.ParseIP(olt.IPAddress)
	if ip == nil || ip.To4() == nil {
		return
	}
	// Use /32 host route — precise, no subnet assumption
	hostRoute := olt.IPAddress + "/32"
	gateway := olt.Router.VPNIP

	if s.vpnManager != nil {
		if err := s.vpnManager.AddPeerAllowedIP(gateway, hostRoute); err != nil {
			log.Printf("[OLT] WireGuard AllowedIP %s → peer %s: %v", hostRoute, gateway, err)
		}
	}

	out, _ := exec.Command("ip", "route", "show", hostRoute).Output()
	if strings.Contains(string(out), gateway) {
		return
	}
	if err := exec.Command("ip", "route", "add", hostRoute, "via", gateway, "dev", "wg0").Run(); err != nil {
		log.Printf("[OLT] route add %s via %s: %v", hostRoute, gateway, err)
		return
	}
	log.Printf("[OLT] route added %s via %s (%s)", hostRoute, gateway, olt.Name)
}

// removeRoute cleans up both the OS route and WireGuard AllowedIP for a deleted OLT.
// Best-effort: failures are logged but do not propagate — the DB record is already gone.
func (s *OLTService) removeRoute(olt *model.OLT) {
	if olt.Router == nil || olt.Router.VPNIP == "" || olt.IPAddress == "" {
		return
	}
	hostRoute := olt.IPAddress + "/32"
	gateway := olt.Router.VPNIP

	// 1. Remove OS route
	if err := exec.Command("ip", "route", "del", hostRoute, "via", gateway, "dev", "wg0").Run(); err != nil {
		log.Printf("[OLT] route del %s: %v (may already be absent)", hostRoute, err)
	} else {
		log.Printf("[OLT] route removed %s (%s)", hostRoute, olt.Name)
	}

	// 2. Remove WireGuard AllowedIP so the peer stops accepting traffic for this host
	if s.vpnManager != nil {
		if err := s.vpnManager.RemovePeerAllowedIP(gateway, hostRoute); err != nil {
			log.Printf("[OLT] WireGuard AllowedIP remove %s from peer %s: %v", hostRoute, gateway, err)
		}
	}
}

// RestoreAllRoutes re-provisions host routes for all active OLTs after a server restart.
func (s *OLTService) RestoreAllRoutes(ctx context.Context) {
	olts, err := s.oltRepo.ListAllWithRouter(ctx)
	if err != nil {
		log.Printf("[OLT] RestoreAllRoutes: list error: %v", err)
		return
	}
	count := 0
	for i := range olts {
		if olts[i].Router != nil && olts[i].Router.VPNIP != "" {
			s.provisionRoute(&olts[i])
			count++
		}
	}
	if count > 0 {
		log.Printf("[OLT] Restored %d route(s) after startup", count)
	}
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
	if err := s.oltRepo.Delete(ctx, tenantID, oltID); err != nil {
		return err
	}
	s.removeRoute(olt) // synchronous — ensures cleanup completes before returning
	return nil
}

func (s *OLTService) List(ctx context.Context, tenantID string, filter repository.OLTFilter) ([]model.OLT, int, error) {
	return s.oltRepo.List(ctx, tenantID, filter)
}

// SyncFromSNMP updates OLT record fields from live SNMP data.
func (s *OLTService) SyncFromSNMP(ctx context.Context, tenantID, oltID string, result *OLTMonitorResult) error {
	olt, err := s.oltRepo.FindByID(ctx, tenantID, oltID)
	if err != nil || olt == nil {
		return ErrOLTNotFound
	}

	// Update fields that come from SNMP: vendor/model parsed from sysDescr, system name
	if result.System.Name != "" && (olt.Model == nil || *olt.Model == "") {
		name := result.System.Name
		olt.Model = &name
	}
	if result.System.Description != "" && (olt.Vendor == nil || *olt.Vendor == "") {
		desc := result.System.Description
		olt.Vendor = &desc
	}

	return s.oltRepo.Update(ctx, olt)
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
