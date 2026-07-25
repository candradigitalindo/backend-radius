package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Xendit Invoice API v2
// Docs: https://developers.xendit.co/api-reference/#create-invoice
// Auth: HTTP Basic Auth with secret key as username, empty password.
// Webhook: x-callback-token header verified against webhook verification token.

const (
	xenditBaseURL    = "https://api.xendit.co"
	xenditBaseURLDev = "https://api.xendit.co" // Xendit uses the same URL; sandbox/live is determined by the API key prefix (xnd_development_ vs xnd_production_)
)

// XenditClient is an HTTP client for the Xendit Invoice API.
type XenditClient struct {
	secretKey    string
	webhookToken string // Webhook Verification Token from Xendit Dashboard → Settings → Callbacks
	baseURL      string
	httpClient   *http.Client
}

// NewXenditClient creates a new Xendit client.
// secretKey: Xendit secret API key (xnd_development_... or xnd_production_...).
// webhookToken: Callback verification token from Xendit Dashboard.
// sandbox parameter is kept for interface consistency but Xendit determines
// sandbox/production based on the API key prefix.
func NewXenditClient(secretKey, webhookToken string, sandbox bool) *XenditClient {
	return &XenditClient{
		secretKey:    secretKey,
		webhookToken: webhookToken,
		baseURL:      xenditBaseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

// TestConnection verifies that the Xendit secret key is valid by calling the balance endpoint.
// Returns nil if the credentials are valid, or an error with a descriptive message.
func (c *XenditClient) TestConnection(ctx context.Context) error {
	url := c.baseURL + "/balance"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// Xendit uses HTTP Basic Auth: secret_key as username, empty password
	req.SetBasicAuth(c.secretKey, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tidak dapat terhubung ke Xendit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Xendit: Secret Key tidak valid (401 Unauthorized)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("Xendit: server merespons dengan status %d", resp.StatusCode)
	}
	return nil
}

// --- Create Invoice ---

// XenditCreateInvoiceRequest holds the parameters for creating a Xendit Invoice.
// Reference: https://developers.xendit.co/api-reference/#create-invoice
type XenditCreateInvoiceRequest struct {
	ExternalID      string `json:"external_id"`
	Amount          int64  `json:"amount"`
	Description     string `json:"description"`
	PayerEmail      string `json:"payer_email,omitempty"`
	CustomerName    string `json:"-"` // mapped to customer object
	CustomerPhone   string `json:"-"` // mapped to customer object
	SuccessRedirect string `json:"success_redirect_url,omitempty"`
	FailureRedirect string `json:"failure_redirect_url,omitempty"`
	CallbackURL     string `json:"-"`                          // set via header X-CALLBACK-URL
	InvoiceDuration int64  `json:"invoice_duration,omitempty"` // seconds, default 86400 (24h)
}

// XenditCreateInvoiceResponse is the response from the Xendit Create Invoice API.
type XenditCreateInvoiceResponse struct {
	ID         string `json:"id"`
	ExternalID string `json:"external_id"`
	InvoiceURL string `json:"invoice_url"`
	Status     string `json:"status"`
	Amount     int64  `json:"amount"`
	ExpiryDate string `json:"expiry_date"` // ISO 8601
}

// XenditCallbackPayload is the webhook notification body from Xendit.
// Reference: https://developers.xendit.co/api-reference/#invoice-callback
type XenditCallbackPayload struct {
	ID                 string  `json:"id"`
	ExternalID         string  `json:"external_id"`
	UserID             string  `json:"user_id"`
	Status             string  `json:"status"` // PAID, EXPIRED
	Amount             float64 `json:"amount"`
	PaidAmount         float64 `json:"paid_amount"`
	PaymentMethod      string  `json:"payment_method"`
	PaymentChannel     string  `json:"payment_channel"`
	PaymentDestination string  `json:"payment_destination"`
	PaidAt             string  `json:"paid_at,omitempty"`
	MerchantName       string  `json:"merchant_name"`
	Currency           string  `json:"currency"`
}

// CreateInvoice creates a Xendit Invoice and returns the payment URL.
func (c *XenditClient) CreateInvoice(ctx context.Context, req XenditCreateInvoiceRequest) (*XenditCreateInvoiceResponse, error) {
	if req.InvoiceDuration == 0 {
		req.InvoiceDuration = 86400 // 24 hours
	}

	body := map[string]interface{}{
		"external_id":      req.ExternalID,
		"amount":           req.Amount,
		"description":      req.Description,
		"invoice_duration": req.InvoiceDuration,
	}

	if req.PayerEmail != "" {
		body["payer_email"] = req.PayerEmail
	}
	if req.SuccessRedirect != "" {
		body["success_redirect_url"] = req.SuccessRedirect
	}
	if req.FailureRedirect != "" {
		body["failure_redirect_url"] = req.FailureRedirect
	}

	// Add customer object if name or phone provided
	if req.CustomerName != "" || req.CustomerPhone != "" {
		customer := map[string]interface{}{}
		if req.CustomerName != "" {
			customer["given_names"] = req.CustomerName
		}
		if req.CustomerPhone != "" {
			customer["mobile_number"] = req.CustomerPhone
		}
		if req.PayerEmail != "" {
			customer["email"] = req.PayerEmail
		}
		body["customer"] = customer
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v2/invoices", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.secretKey+":")))
	if req.CallbackURL != "" {
		httpReq.Header.Set("X-CALLBACK-URL", req.CallbackURL)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("xendit create invoice: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("xendit read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("xendit error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result XenditCreateInvoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("xendit decode response: %w", err)
	}

	return &result, nil
}

// VerifyWebhookToken validates the x-callback-token header from a Xendit webhook.
// Xendit sends the Callback Verification Token in the "x-callback-token" header.
// This must match the token configured in the Xendit Dashboard.
// Reference: https://developers.xendit.co/api-reference/#webhook-verification
func (c *XenditClient) VerifyWebhookToken(callbackToken string) bool {
	if c.webhookToken == "" || callbackToken == "" {
		return false
	}
	// Use HMAC comparison to prevent timing attacks even though it's a simple string match
	return hmac.Equal([]byte(c.webhookToken), []byte(callbackToken))
}

// IsXenditPaymentSuccess returns true if the callback status indicates successful payment.
func IsXenditPaymentSuccess(payload XenditCallbackPayload) bool {
	return payload.Status == "PAID" || payload.Status == "SETTLED"
}

// IsXenditPaymentFailed returns true if the callback status indicates a failed/expired payment.
func IsXenditPaymentFailed(payload XenditCallbackPayload) bool {
	return payload.Status == "EXPIRED"
}
