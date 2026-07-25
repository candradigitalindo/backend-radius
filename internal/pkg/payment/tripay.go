package payment

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	tripayBaseURL    = "https://tripay.co.id/api"
	tripayBaseURLDev = "https://tripay.co.id/api-sandbox"
)

// TripayClient is an HTTP client for the Tripay payment gateway.
type TripayClient struct {
	apiKey     string
	privateKey string
	merchantID string
	baseURL    string
	httpClient *http.Client
}

// NewTripayClient creates a new Tripay client.
func NewTripayClient(apiKey, privateKey, merchantID string, sandbox bool) *TripayClient {
	baseURL := tripayBaseURL
	if sandbox {
		baseURL = tripayBaseURLDev
	}
	return &TripayClient{
		apiKey:     apiKey,
		privateKey: privateKey,
		merchantID: merchantID,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateTransactionRequest holds the parameters for creating a Tripay transaction.
type CreateTransactionRequest struct {
	Method        string `json:"method"`
	MerchantRef   string `json:"merchant_ref"`
	Amount        int64  `json:"amount"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
	CustomerPhone string `json:"customer_phone"`
	ReturnURL     string `json:"return_url,omitempty"`
	CallbackURL   string `json:"callback_url,omitempty"`
	ExpiredTime   int64  `json:"expired_time"`
}

// CreateTransactionResponse is the response from the create transaction API.
type CreateTransactionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Reference   string `json:"reference"`
		MerchantRef string `json:"merchant_ref"`
		PaymentURL  string `json:"payment_url"`
		CheckoutURL string `json:"checkout_url"`
		Status      string `json:"status"`
		Amount      int64  `json:"amount"`
		ExpiredTime int64  `json:"expired_time"`
	} `json:"data"`
}

// TransactionStatusResponse is the response from the transaction detail API.
type TransactionStatusResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Reference   string `json:"reference"`
		MerchantRef string `json:"merchant_ref"`
		Status      string `json:"status"`
		Amount      int64  `json:"amount"`
	} `json:"data"`
}

// TripayCallbackPayload is the webhook notification body from Tripay.
type TripayCallbackPayload struct {
	Reference       string `json:"reference"`
	MerchantRef     string `json:"merchant_ref"`
	PaymentMethod   string `json:"payment_method"`
	PaymentSelect   string `json:"payment_method_code"`
	TotalAmount     int64  `json:"total_amount"`
	FeeMerchant     int64  `json:"fee_merchant"`
	AmountReceived  int64  `json:"amount_received"`
	IsClosedPayment int    `json:"is_closed_payment"`
	Status          string `json:"status"`
	PaidAt          *int64 `json:"paid_at,omitempty"`
}

// CreateTransaction creates a closed payment transaction in Tripay.
func (c *TripayClient) CreateTransaction(ctx context.Context, req CreateTransactionRequest) (*CreateTransactionResponse, error) {
	req.MerchantRef = c.merchantID + "-" + req.MerchantRef
	req.ExpiredTime = time.Now().Add(24 * time.Hour).Unix()

	// Generate signature: HMAC-SHA256(privateKey, merchantID+merchantRef+amount)
	sig := c.generateSignature(req.MerchantRef, req.Amount)

	body := map[string]interface{}{
		"method":         req.Method,
		"merchant_ref":   req.MerchantRef,
		"amount":         req.Amount,
		"customer_name":  req.CustomerName,
		"customer_email": req.CustomerEmail,
		"customer_phone": req.CustomerPhone,
		"expired_time":   req.ExpiredTime,
		"signature":      sig,
	}
	if req.ReturnURL != "" {
		body["return_url"] = req.ReturnURL
	}
	if req.CallbackURL != "" {
		body["callback_url"] = req.CallbackURL
	}

	var result CreateTransactionResponse
	if err := c.post(ctx, "/transaction/create", body, &result); err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, fmt.Errorf("tripay: %s", result.Message)
	}
	return &result, nil
}

// TestConnection verifies that the Tripay API key is valid by calling the merchant profile endpoint.
// Returns nil if the credentials are valid, or an error with a descriptive message.
func (c *TripayClient) TestConnection(ctx context.Context) error {
	url := c.baseURL + "/merchant/profile"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tidak dapat terhubung ke Tripay: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("respons Tripay tidak valid: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("Tripay: %s", result.Message)
	}
	return nil
}

// VerifyCallbackSignature validates the X-Callback-Signature header that Tripay
// sends with every callback request. Tripay computes the signature as
// HMAC-SHA256 over the *entire raw JSON request body* using the merchant private
// key — NOT over the create-transaction string and NOT from a body field.
// Docs: https://tripay.co.id/developer?tab=callback
func (c *TripayClient) VerifyCallbackSignature(rawBody []byte, signature string) bool {
	if signature == "" || c.privateKey == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.privateKey))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func (c *TripayClient) generateSignature(merchantRef string, amount int64) string {
	mac := hmac.New(sha256.New, []byte(c.privateKey))
	mac.Write([]byte(c.merchantID + merchantRef + strconv.FormatInt(amount, 10)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (c *TripayClient) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tripay post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("tripay read body: %w", err)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("tripay decode response: %w", err)
	}
	return nil
}
