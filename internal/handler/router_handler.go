package handler

import (
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type RouterHandler struct {
	routerService *service.RouterService
}

func NewRouterHandler(routerService *service.RouterService) *RouterHandler {
	return &RouterHandler{routerService: routerService}
}

// parseRouterPush decodes body dari router (heartbeat / interface-stats) sebagai
// JSON tanpa melihat header Content-Type. RouterOS 6 /tool fetch tetap mengirim
// Content-Type bawaannya sendiri walau script sudah menyetel
// "Content-Type: application/json", sehingga BodyParser memakai decoder yang
// salah dan semua push dari ROS 6 tertolak 400.
//
// Log kegagalan parse di-rate-limit: router rusak yang push tiap 5 detik akan
// membanjiri log (±17 ribu baris/hari) tanpa ini.
var (
	parsePushLogMu   sync.Mutex
	parsePushLastLog time.Time
)

func parseRouterPush(c *fiber.Ctx, out interface{}) error {
	if err := json.Unmarshal(c.Body(), out); err != nil {
		parsePushLogMu.Lock()
		if time.Since(parsePushLastLog) > 30*time.Second {
			parsePushLastLog = time.Now()
			body := c.Body()
			if len(body) > 300 {
				body = body[:300]
			}
			log.Printf("router push parse error from %s ct=%q body=%q", c.IP(), c.Get("Content-Type"), body)
		}
		parsePushLogMu.Unlock()
		return err
	}
	return nil
}

func (h *RouterHandler) routerResponse(router interface{}) fiber.Map {
	result := fiber.Map{
		"data":              router,
		"server_public_ip":  h.routerService.GetServerPublicIP(),
		"server_public_key": h.routerService.GetServerPublicKey(),
	}
	if r, ok := router.(*model.Router); ok && r.VPNIP != "" {
		status := h.routerService.GetVPNStatus(r.VPNIP)
		result["vpn_status"] = status
		if status.Connected {
			r.IsOnline = true
		}
	}
	return result
}

type createRouterRequest struct {
	Name          string `json:"name"`
	RouterType    string `json:"router_type"`
	SNMPCommunity string `json:"snmp_community"`
}

type updateRouterRequest struct {
	Name          string `json:"name"`
	RouterType    string `json:"router_type"`
	SNMPCommunity string `json:"snmp_community"`
	IsActive      bool   `json:"is_active"`
}

type heartbeatRequest struct {
	Token       string `json:"token"`
	Identity    string `json:"identity"`
	RouterOSVer string `json:"router_os_ver"`
	BoardName   string `json:"board_name"`
	Uptime      string `json:"uptime"`
	CPULoad     int    `json:"cpu_load"`
	FreeMemory  int64  `json:"free_memory"`
	TotalMemory int64  `json:"total_memory"`
}

type registerVPNKeyRequest struct {
	PublicKey string `json:"public_key"`
}

func (h *RouterHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	var req createRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama router wajib diisi"})
	}
	router, err := h.routerService.Create(c.Context(), service.CreateRouterInput{
		TenantID:      tenantID,
		Name:          req.Name,
		RouterType:    req.RouterType,
		SNMPCommunity: req.SNMPCommunity,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal membuat router"})
	}
	return c.Status(fiber.StatusCreated).JSON(h.routerResponse(router))
}

func (h *RouterHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	router, err := h.routerService.GetByID(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}

	resp := h.routerResponse(router)
	if interfaces := c.Query("interfaces"); interfaces != "" {
		ifaces := strings.Split(interfaces, ",")
		traffic, err := h.routerService.GetRealtimeInterfaceTraffic(c.Context(), router, ifaces)
		if err == nil {
			resp["realtime_bandwidth"] = traffic
		}
	}
	return c.JSON(resp)
}

// PushInterfaceStats receives interface counters pushed by a router (token-auth).
// This is the VPN-free path: the router posts outbound, server derives throughput.
func (h *RouterHandler) PushInterfaceStats(c *fiber.Ctx) error {
	var req struct {
		Token      string                   `json:"token"`
		Interfaces []service.IfaceStatInput `json:"interfaces"`
	}
	if err := parseRouterPush(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token wajib diisi"})
	}
	if err := h.routerService.RecordInterfaceStats(c.Context(), req.Token, req.Interfaces); err != nil {
		if errors.Is(err, service.ErrInvalidHeartbeat) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token tidak valid"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal menyimpan statistik interface"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// GetInterfaceStats returns recent pushed interface throughput for the chart.
func (h *RouterHandler) GetInterfaceStats(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	minutes, _ := strconv.Atoi(c.Query("minutes", "10"))

	stats, err := h.routerService.GetInterfaceStats(c.Context(), tenantID, routerID, minutes)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat statistik interface"})
	}
	return c.JSON(fiber.Map{"data": stats})
}

func (h *RouterHandler) GetInterfaces(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	interfaces, err := h.routerService.GetInterfaces(c.Context(), tenantID, routerID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(interfaces)
}

func (h *RouterHandler) Update(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	var req updateRouterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama router wajib diisi"})
	}
	router, err := h.routerService.Update(c.Context(), tenantID, routerID, service.UpdateRouterInput{
		Name:          req.Name,
		RouterType:    req.RouterType,
		SNMPCommunity: req.SNMPCommunity,
		IsActive:      req.IsActive,
	})
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(h.routerResponse(router))
}

func (h *RouterHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	if err := h.routerService.Delete(c.Context(), tenantID, routerID); err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"message": "Router berhasil dihapus"})
}

func (h *RouterHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	filter := repository.RouterFilter{
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		Page:    page,
		PerPage: perPage,
	}
	routers, total, err := h.routerService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar router"})
	}
	for i := range routers {
		if routers[i].VPNIP != "" {
			status := h.routerService.GetVPNStatus(routers[i].VPNIP)
			if status.Connected {
				routers[i].IsOnline = true
			}
		}
	}
	return c.JSON(fiber.Map{
		"data":              routers,
		"total":             total,
		"page":              page,
		"per_page":          perPage,
		"server_public_ip":  h.routerService.GetServerPublicIP(),
		"server_public_key": h.routerService.GetServerPublicKey(),
	})
}

func (h *RouterHandler) RegenerateToken(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	token, err := h.routerService.RegenerateToken(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
	return c.JSON(fiber.Map{"heartbeat_token": token})
}

func (h *RouterHandler) Heartbeat(c *fiber.Ctx) error {
	var req heartbeatRequest
	if err := parseRouterPush(c, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Token wajib diisi"})
	}
	// Real router IP: behind nginx/Cloudflare, c.IP() is the proxy. The original
	// client (router public IP) is the first entry of X-Forwarded-For.
	senderIP := strings.TrimSpace(c.IP())
	if ips := c.IPs(); len(ips) > 0 && strings.TrimSpace(ips[0]) != "" {
		senderIP = strings.TrimSpace(ips[0])
	}

	err := h.routerService.Heartbeat(c.Context(), req.Token, repository.HeartbeatInfo{
		Identity:    req.Identity,
		RouterOSVer: req.RouterOSVer,
		BoardName:   req.BoardName,
		Uptime:      req.Uptime,
		CPULoad:     req.CPULoad,
		FreeMemory:  req.FreeMemory,
		TotalMemory: req.TotalMemory,
	}, senderIP)
	if err != nil {
		if errors.Is(err, service.ErrInvalidHeartbeat) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Heartbeat failed"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *RouterHandler) ListSessions(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	activeOnly := c.Query("status") == "active"
	sessions, total, err := h.routerService.ListSessions(c.Context(), tenantID, routerID, activeOnly, page, perPage)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to list sessions"})
	}
	return c.JSON(fiber.Map{
		"data":     sessions,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *RouterHandler) GetTraffic(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	traffic, err := h.routerService.GetTraffic(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get traffic"})
	}
	return c.JSON(traffic)
}

func (h *RouterHandler) TestConnection(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	reachable, err := h.routerService.TestConnection(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Test failed"})
	}
	return c.JSON(fiber.Map{"reachable": reachable})
}

func (h *RouterHandler) SyncSessions(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	cleaned, err := h.routerService.SyncSessions(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sync failed"})
	}
	return c.JSON(fiber.Map{"cleaned_sessions": cleaned})
}

func (h *RouterHandler) ListVPNPeers(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	peers, err := h.routerService.ListVPNPeers(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"peers": peers})
}

func (h *RouterHandler) RegisterVPNKey(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	var req registerVPNKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}
	if req.PublicKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Public key WireGuard wajib diisi"})
	}
	router, err := h.routerService.RegisterVPNKey(c.Context(), tenantID, routerID, req.PublicKey)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(h.routerResponse(router))
}

// EnableLegacyVPN provisions L2TP/SSTP tunnel credentials (RouterOS 6 / router
// di belakang NAT). Idempotent — memanggil ulang mengembalikan kredensial yang sama.
func (h *RouterHandler) EnableLegacyVPN(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	router, err := h.routerService.EnableLegacyVPN(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(h.routerResponse(router))
}

func (h *RouterHandler) GetMikroTikConfig(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	config, err := h.routerService.GetMikroTikConfig(c.Context(), tenantID, routerID)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat konfigurasi MikroTik"})
	}
	return c.JSON(config)
}

func (h *RouterHandler) ListConnectionLogs(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	routerID := c.Params("id")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))
	logs, total, err := h.routerService.ListConnectionLogs(c.Context(), tenantID, routerID, page, perPage)
	if err != nil {
		if errors.Is(err, service.ErrRouterNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat histori koneksi"})
	}
	return c.JSON(fiber.Map{
		"data":     logs,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}
