package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/candrasyahputra/radius-server/internal/config"
)

type Client struct {
	baseURL    string
	apiSecret  string
	httpClient *http.Client
}

func NewClient(cfg config.WhatsAppConfig) *Client {
	return &Client{
		baseURL:   cfg.ServiceURL,
		apiSecret: cfg.APISecret,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// SessionStatus represents the WA session status response.
type SessionStatus struct {
	Status  string      `json:"status"`
	QR      string      `json:"qr,omitempty"`
	Message string      `json:"message,omitempty"`
	Device  *DeviceInfo `json:"device,omitempty"`
}

// DeviceInfo represents the connected WhatsApp device.
type DeviceInfo struct {
	Phone       string `json:"phone"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	ConnectedAt string `json:"connectedAt"`
}

// SendResult represents the result of sending a single message.
type SendResult struct {
	ID        string `json:"id,omitempty"`
	JID       string `json:"jid,omitempty"`
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
}

// BroadcastResult represents the result of a broadcast operation.
type BroadcastResult struct {
	Success       int          `json:"success"`
	Failed        int          `json:"failed"`
	Pending       int          `json:"pending"`
	PendingPhones []string     `json:"pendingPhones"`
	Details       []SendDetail `json:"details"`
}

type SendDetail struct {
	Phone  string `json:"phone"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ReminderItem represents a single reminder to send.
type ReminderItem struct {
	Phone         string `json:"phone"`
	CustomerName  string `json:"customerName"`
	InvoiceNumber string `json:"invoiceNumber"`
	Amount        int64  `json:"amount"`
	DueDate       string `json:"dueDate"`
	Message       string `json:"message"`
}

// StartSession starts/connects a WA session for a tenant.
func (c *Client) StartSession(ctx context.Context, tenantID string) (*SessionStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/sessions/%s/start", tenantID), nil)
	if err != nil {
		return nil, err
	}
	var result SessionStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// GetSessionStatus checks WA connection status for a tenant.
func (c *Client) GetSessionStatus(ctx context.Context, tenantID string) (*SessionStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/sessions/%s/status", tenantID), nil)
	if err != nil {
		return nil, err
	}
	var result SessionStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// GetQR retrieves the QR code for pairing.
func (c *Client) GetQR(ctx context.Context, tenantID string) (*SessionStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/sessions/%s/qr", tenantID), nil)
	if err != nil {
		return nil, err
	}
	var result SessionStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// StopSession disconnects and cleans up a WA session.
func (c *Client) StopSession(ctx context.Context, tenantID string) (*SessionStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/sessions/%s", tenantID), nil)
	if err != nil {
		return nil, err
	}
	var result SessionStatus
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// SendMessage sends a single WhatsApp message.
func (c *Client) SendMessage(ctx context.Context, tenantID, phone, message string) (*SendResult, error) {
	return c.SendMessageWithFile(ctx, tenantID, phone, message, "")
}

// SendMessageWithFile sends a single WhatsApp message with an optional file (image or document).
func (c *Client) SendMessageWithFile(ctx context.Context, tenantID, phone, message, fileURL string) (*SendResult, error) {
	body := map[string]interface{}{
		"tenantId": tenantID,
		"phone":    phone,
		"message":  message,
	}

	if fileURL != "" {
		ext := strings.ToLower(filepath.Ext(fileURL))
		// Strip query params if any
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}

		isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif"

		if isImage {
			body["image"] = fileURL
		} else {
			body["document"] = fileURL
			// Extract filename from URL
			parts := strings.Split(fileURL, "/")
			fileName := parts[len(parts)-1]
			if idx := strings.Index(fileName, "?"); idx != -1 {
				fileName = fileName[:idx]
			}
			body["fileName"] = fileName
		}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/messages/send", body)
	if err != nil {
		return nil, err
	}
	var result SendResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// SendBroadcast sends messages to multiple phone numbers.
func (c *Client) SendBroadcast(ctx context.Context, tenantID string, phones []string, message string) (*BroadcastResult, error) {
	return c.SendBroadcastWithFile(ctx, tenantID, phones, message, "")
}

// SendBroadcastWithImage is kept for backward compatibility but calls SendBroadcastWithFile.
func (c *Client) SendBroadcastWithImage(ctx context.Context, tenantID string, phones []string, message, imageURL string) (*BroadcastResult, error) {
	return c.SendBroadcastWithFile(ctx, tenantID, phones, message, imageURL)
}

// SendBroadcastWithFile sends messages with an optional file (image or document) to multiple phone numbers.
func (c *Client) SendBroadcastWithFile(ctx context.Context, tenantID string, phones []string, message, fileURL string) (*BroadcastResult, error) {
	body := map[string]interface{}{
		"tenantId": tenantID,
		"phones":   phones,
		"message":  message,
	}

	if fileURL != "" {
		ext := strings.ToLower(filepath.Ext(fileURL))
		// Strip query params if any
		if idx := strings.Index(ext, "?"); idx != -1 {
			ext = ext[:idx]
		}

		isImage := ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif"

		if isImage {
			body["image"] = fileURL
		} else {
			body["document"] = fileURL
			// Extract filename from URL
			parts := strings.Split(fileURL, "/")
			fileName := parts[len(parts)-1]
			if idx := strings.Index(fileName, "?"); idx != -1 {
				fileName = fileName[:idx]
			}
			body["fileName"] = fileName
		}
	}

	resp, err := c.doRequest(ctx, http.MethodPost, "/api/messages/broadcast", body)
	if err != nil {
		return nil, err
	}
	var result BroadcastResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// SendReminders sends formatted invoice reminders.
func (c *Client) SendReminders(ctx context.Context, tenantID string, reminders []ReminderItem) (*BroadcastResult, error) {
	body := map[string]interface{}{
		"tenantId":  tenantID,
		"reminders": reminders,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/messages/reminder", body)
	if err != nil {
		return nil, err
	}
	var result BroadcastResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("whatsapp service error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
