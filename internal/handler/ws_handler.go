package handler

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/candrasyahputra/radius-server/internal/service"
)

type WSHandler struct {
	dashboardService *service.DashboardService
	bandwidthService *service.BandwidthService
	routerService    *service.RouterService
	hub              *WSHub
	publicKey        *rsa.PublicKey
}

func NewWSHandler(dashboardService *service.DashboardService, bandwidthService *service.BandwidthService, routerService *service.RouterService, publicKeyPath string) *WSHandler {
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		panic("ws: failed to read JWT public key: " + err.Error())
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
	if err != nil {
		panic("ws: failed to parse JWT public key: " + err.Error())
	}

	h := &WSHandler{
		dashboardService: dashboardService,
		bandwidthService: bandwidthService,
		routerService:    routerService,
		hub:              NewWSHub(),
		publicKey:        pubKey,
	}
	go h.broadcastLoop()
	return h
}

type WSHub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]bool
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
}

func (h *WSHub) Register(tenantID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[*websocket.Conn]bool)
	}
	h.clients[tenantID][conn] = true
}

func (h *WSHub) Unregister(tenantID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[tenantID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(h.clients, tenantID)
		}
	}
}

func (h *WSHub) GetTenants() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	tenants := make([]string, 0, len(h.clients))
	for t := range h.clients {
		tenants = append(tenants, t)
	}
	return tenants
}

func (h *WSHub) Broadcast(tenantID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients[tenantID] {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			conn.Close()
		}
	}
}

func (h *WSHandler) UpgradeCheck(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token tidak ditemukan"})
		}

		parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok || t.Method.Alg() != "RS256" {
				return nil, jwt.ErrSignatureInvalid
			}
			return h.publicKey, nil
		})
		if err != nil || !parsed.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Token tidak valid"})
		}

		claims, ok := parsed.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token claims"})
		}

		tenantID, _ := claims["tenant_id"].(string)
		if tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Tenant tidak ditemukan dalam token"})
		}

		c.Locals("tenant_id", tenantID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *WSHandler) Handle() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		tenantID, _ := c.Locals("tenant_id").(string)
		if tenantID == "" {
			c.Close()
			return
		}

		h.hub.Register(tenantID, c)
		defer h.hub.Unregister(tenantID, c)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				break
			}

			var req struct {
				Action     string   `json:"action"`
				RouterID   string   `json:"router_id"`
				Interfaces []string `json:"interfaces"`
			}
			if err := json.Unmarshal(msg, &req); err == nil {
				if req.Action == "monitor_router" {
					go h.startRouterMonitor(ctx, c, tenantID, req.RouterID, req.Interfaces)
				}
			}
		}
	})
}

func (h *WSHandler) startRouterMonitor(ctx context.Context, c *websocket.Conn, tenantID, routerID string, interfaces []string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	router, err := h.routerService.GetByID(ctx, tenantID, routerID)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			traffic, err := h.routerService.GetRealtimeInterfaceTraffic(ctx, router, interfaces)
			if err != nil {
				continue
			}

			msg := wsMessage{
				Type: "router_bandwidth",
				Data: fiber.Map{
					"router_id": routerID,
					"traffic":   traffic,
				},
				Time: time.Now().Unix(),
			}

			data, _ := json.Marshal(msg)
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		}
	}
}

type wsMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Time int64       `json:"time"`
}

func (h *WSHandler) broadcastLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		tenants := h.hub.GetTenants()
		now := time.Now()

		for _, tenantID := range tenants {
			ctx := context.Background()

			stats, err := h.dashboardService.GetStats(
				ctx, tenantID,
				int(now.Month()), now.Year(),
			)
			if err != nil {
				log.Printf("[ws] error getting stats for tenant %s: %v", tenantID, err)
				continue
			}

			msg := wsMessage{
				Type: "dashboard_stats",
				Data: stats,
				Time: now.Unix(),
			}

			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			h.hub.Broadcast(tenantID, data)

			bw, err := h.bandwidthService.GetUsageSummary(
				ctx, tenantID,
				int(now.Month()), now.Year(),
			)
			if err == nil {
				bwMsg := wsMessage{
					Type: "bandwidth_summary",
					Data: bw,
					Time: now.Unix(),
				}
				if bwData, err := json.Marshal(bwMsg); err == nil {
					h.hub.Broadcast(tenantID, bwData)
				}
			}
		}
	}
}
