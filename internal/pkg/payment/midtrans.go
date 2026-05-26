package payment

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	midtransSnapURL    = "https://app.midtrans.com/snap/v1/transactions"
	midtransSnapURLDev = "https://app.sandbox.midtrans.com/snap/v1/transactions"
)

// MidtransClient is an HTTP client for the Midtrans Snap API.
type MidtransClient struct {
	serverKey  string
	baseURL    string
	httpClient *http.Client
}

// NewMidtransClient creates a new Midtrans Snap client.
func NewMidtransClient(serverKey string, sandbox bool) *MidtransClient {
	baseURL := midtransSnapURL
	if sandbox {
		baseURL = midtransSnapURLDev
	}
	return &MidtransClient{
		serverKey:  serverKey,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// TestConnection verifies that the Midtrans server key is valid by calling the payment methods endpoint.
// Returns nil if the credentials are valid, or an error with a descriptive message.
func (c *MidtransClient) TestConnection(ctx context.Context) error {
	// Use Midtrans Core API status endpoint — a lightweight authenticated call
	statusURL := "https://api.midtrans.com/v2/payment-types"
	if c.baseURL == midtransSnapURLDev {
		statusURL = "https://api.sandbox.midtrans.com/v2/payment-types"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return err
	}
	// Midtrans uses HTTP Basic Auth: server_key as username, empty password
	req.SetBasicAuth(c.serverKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tidak dapat terhubung ke Midtrans: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Midtrans: Server Key tidak valid (401 Unauthorized)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Midtrans: server merespons dengan status %d", resp.StatusCode)
	}
	return nil
}

// MidtransCreateRequest holds parameters for creating a Midtrans Snap transaction.
type MidtransCreateRequest struct {
	OrderID         string
	GrossAmount     int64
	FirstName       string
	Email           string
	Phone           string
	FinishURL       string
	ErrorURL        string
	ItemName        string
	NotificationURL string // server-to-server webhook URL
}

// MidtransCreateResponse is the response from the Snap create transaction API.
type MidtransCreateResponse struct {
	Token         string   `json:"token"`
	RedirectURL   string   `json:"redirect_url"`
	ErrorMessages []string `json:"error_messages,omitempty"`
}

// MidtransNotification is the webhook notification body sent by Midtrans.
type MidtransNotification struct {
	TransactionID     string `json:"transaction_id"`
	OrderID           string `json:"order_id"`
	GrossAmount       string `json:"gross_amount"`
	PaymentType       string `json:"payment_type"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
}

// CreateTransaction creates a Midtrans Snap transaction and returns the payment token and URL.
func (c *MidtransClient) CreateTransaction(ctx context.Context, req MidtransCreateRequest) (*MidtransCreateResponse, error) {
	body := map[string]interface{}{
		"transaction_details": map[string]interface{}{
			"order_id":     req.OrderID,
			"gross_amount": req.GrossAmount,
		},
		"customer_details": map[string]interface{}{
			"first_name": req.FirstName,
			"email":      req.Email,
			"phone":      req.Phone,
		},
		"item_details": []map[string]interface{}{
			{
				"id":       req.OrderID,
				"price":    req.GrossAmount,
				"quantity": 1,
				"name":     req.ItemName,
			},
		},
	}

	if req.FinishURL != "" {
		body["callbacks"] = map[string]string{
			"finish": req.FinishURL,
			"error":  req.ErrorURL,
		}
	}


	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.serverKey+":")))
	if req.NotificationURL != "" {
		httpReq.Header.Set("X-Override-Notification", req.NotificationURL)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("midtrans create transaction: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("midtrans read body: %w", err)
	}

	var result MidtransCreateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("midtrans decode response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		msg := "midtrans error"
		if len(result.ErrorMessages) > 0 {
			msg = result.ErrorMessages[0]
		}
		return nil, fmt.Errorf("midtrans: %s (status %d)", msg, resp.StatusCode)
	}

	return &result, nil
}

// VerifyWebhookSignature validates the signature from a Midtrans webhook notification.
// Signature = SHA512(order_id + status_code + gross_amount + server_key)
func (c *MidtransClient) VerifyWebhookSignature(n MidtransNotification) bool {
	raw := n.OrderID + n.StatusCode + n.GrossAmount + c.serverKey
	hash := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(hash[:])
	return expected == n.SignatureKey
}

// IsPaymentSuccess returns true if the notification status indicates a successful payment.
func IsPaymentSuccess(n MidtransNotification) bool {
	switch n.TransactionStatus {
	case "settlement":
		return true
	case "capture":
		return n.FraudStatus == "accept"
	}
	return false
}

// IsPaymentFailed returns true if the notification status indicates a failed/expired payment.
func IsPaymentFailed(n MidtransNotification) bool {
	switch n.TransactionStatus {
	case "cancel", "expire", "deny":
		return true
	}
	return false
}

// GrossAmountInt64 parses the gross_amount string from the notification to int64.
func GrossAmountInt64(n MidtransNotification) (int64, error) {
	f, err := strconv.ParseFloat(n.GrossAmount, 64)
	if err != nil {
		return 0, fmt.Errorf("parse gross_amount %q: %w", n.GrossAmount, err)
	}
	return int64(f), nil
}
