package handler

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/pkg/genieacs"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

// CWMPHandler proxies TR-069 requests and identifies tenant via PPPoE username in XML body.
type CWMPHandler struct {
	genieacsClient *genieacs.Client
	customerRepo   repository.CustomerRepository
	ontRepo        repository.ONTRepository
	cwmpUpstream   string
	httpClient     *http.Client
}

func NewCWMPHandler(genieacsClient *genieacs.Client, tenantRepo repository.TenantRepository, customerRepo repository.CustomerRepository, ontRepo repository.ONTRepository, cwmpUpstream string) *CWMPHandler {
	return &CWMPHandler{
		genieacsClient: genieacsClient,
		customerRepo:   customerRepo,
		ontRepo:        ontRepo,
		cwmpUpstream:   strings.TrimRight(cwmpUpstream, "/"),
		httpClient:     &http.Client{Timeout: 120 * time.Second},
	}
}

func (h *CWMPHandler) Handle(c *fiber.Ctx) error {
	if c.Method() != fiber.MethodPost {
		return c.Status(fiber.StatusOK).Send(nil)
	}

	body := make([]byte, len(c.Body()))
	copy(body, c.Body())
	log.Printf("DEBUG CWMP BODY: %s", string(body))
log.Printf("DEBUG CWMP BODY: %s", string(body))

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, h.cwmpUpstream+"/", bytes.NewReader(body))
	if err != nil {
		log.Printf("CWMP proxy: create request error: %v", err)
		return c.SendStatus(fiber.StatusBadGateway)
	}

	c.Request().Header.VisitAll(func(k, v []byte) {
		key := string(k)
		if !strings.EqualFold(key, "Host") {
			req.Header.Set(key, string(v))
		}
	})
	req.Header.Set("X-Real-IP", c.IP())
	req.Header.Set("X-Forwarded-For", c.IP())

	resp, err := h.httpClient.Do(req)
	if err != nil {
		log.Printf("CWMP proxy: upstream error: %v", err)
		return c.SendStatus(fiber.StatusBadGateway)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	for k, vs := range resp.Header {
		for _, v := range vs {
			c.Set(k, v)
		}
	}

	// Auto-identify tenant via body XML content
	go h.tagDeviceViaUsername(body)

	return c.Status(resp.StatusCode).Send(respBody)
}

func (h *CWMPHandler) tagDeviceViaUsername(body []byte) {
	serial := extractSerialFromInform(body)
	username := extractUsernameFromInform(body)
	if serial == "" || username == "" {
		return
	}

	// Find which tenant this username belongs to
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	customer, err := h.customerRepo.FindByPPPoEUsernameGlobal(ctx, username)
	cancel()

	if err != nil || customer == nil {
		return // username not found in any tenant
	}

	tenantID := customer.TenantID
	tag := "tenant:" + tenantID

	for attempt := 1; attempt <= 10; attempt++ {
		time.Sleep(1 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		device, err := h.genieacsClient.FindDeviceBySerialNumber(ctx, serial)
		cancel()

		if err != nil || device == nil {
			continue
		}

		for _, t := range device.Tags {
			if t == tag { return }
		}

		tagCtx, tagCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = h.genieacsClient.AddTag(tagCtx, device.ID, tag)
		tagCancel()

		log.Printf("CWMP tag: device serial=%s auto-tagged tenant=%s via username=%s", serial, tenantID, username)

		// Link ONT ke customer di database jika belum terhubung
		if h.ontRepo != nil {
			linkCtx, linkCancel := context.WithTimeout(context.Background(), 10*time.Second)
			ont, err := h.ontRepo.FindBySerialNumber(linkCtx, tenantID, serial)
			linkCancel()
			if err == nil && ont != nil && ont.CustomerID == nil {
				linkCtx2, linkCancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				if err := h.ontRepo.LinkToCustomer(linkCtx2, ont.ID, customer.ID, customer.TenantID); err == nil {
					log.Printf("CWMP link: ONT %s dihubungkan ke customer %s (tenant %s)", serial, customer.ID, customer.TenantID)
				} else {
					log.Printf("CWMP link: gagal menghubungkan ONT %s ke customer %s: %v", serial, customer.ID, err)
				}
				linkCancel2()
			}
		}
		return
	}
}

func extractSerialFromInform(body []byte) string {
	s := string(body)
	for _, prefix := range []string{"cwmp:", "SOAP-ENV:", "soap:", "soapenv:", "SOAP:", "env:"} {
		s = strings.ReplaceAll(s, prefix, "")
	}
	const open = "<SerialNumber>"
	const close = "</SerialNumber>"
	idx := strings.Index(s, open)
	if idx < 0 { return "" }
	start := idx + len(open)
	end := strings.Index(s[start:], close)
	if end < 0 { return "" }
	return strings.TrimSpace(s[start : start+end])
}

func extractUsernameFromInform(body []byte) string {
	s := string(body)
	// Try to find username value in typical TR-069 locations (WANDevice.1.WANConnectionDevice.1.WANPPPConnection.1.Username)
	// We search for the pattern <Value ...>...username...</Value> near the word "Username"
	idx := strings.Index(s, "Username")
	if idx > 0 {
		remaining := s[idx:]
		vStart := strings.Index(remaining, ">")
		if vStart > 0 {
			vEnd := strings.Index(remaining[vStart:], "</")
			if vEnd > 0 {
				val := strings.TrimSpace(remaining[vStart+1 : vStart+vEnd])
				// Clean up if it's wrapped in more tags
				if strings.Contains(val, ">") {
					val = val[strings.LastIndex(val, ">")+1:]
				}
				return val
			}
		}
	}
	return ""
}
