package handler

import (
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type CustomerHandler struct {
	customerService *service.CustomerService
	invoiceService  *service.InvoiceService
}

func NewCustomerHandler(customerService *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{customerService: customerService}
}

func (h *CustomerHandler) WithInvoiceService(svc *service.InvoiceService) *CustomerHandler {
	h.invoiceService = svc
	return h
}

type createCustomerRequest struct {
	Name             string   `json:"name"`
	NIK              string   `json:"nik"`
	Phone            string   `json:"phone"`
	Email            string   `json:"email"`
	Address          string   `json:"address"`
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	ConnectionType   string   `json:"connection_type"`
	// PPPoE credentials opsional; kosong = digenerate otomatis oleh sistem
	PPPoEUsername    string   `json:"pppoe_username"`
	PPPoEPassword    string   `json:"pppoe_password"`
	IPAddress        string   `json:"ip_address"`
	PackageID        *string  `json:"package_id"`
	RouterID         *string  `json:"router_id"`
	ODPPortID        *string  `json:"odp_port_id"`
	JoinDate         string   `json:"join_date"`
	BillingType      string   `json:"billing_type"`
	BillingProfileID *string  `json:"billing_profile_id"`
	CustomPrice      *int64   `json:"custom_price"`
	Discount         int64    `json:"discount"`
	AdditionalFee    int64    `json:"additional_fee"`
	FeeDescription   string   `json:"fee_description"`
	Notes            string   `json:"notes"`
	// FTTH / GenieACS: serial number perangkat ONT
	SerialNumber string  `json:"serial_number"`
	ONTVendor    *string `json:"ont_vendor"`
	ONTModel     *string `json:"ont_model"`
	// Kode referral yang digunakan saat mendaftar (opsional)
	ReferralCodeUsed string  `json:"referral_code_used"`
	ResellerID       *string `json:"reseller_id"`
}

type updateCustomerProfileRequest struct {
	Name       string   `json:"name"`
	NIK        string   `json:"nik"`
	Phone      string   `json:"phone"`
	Email      string   `json:"email"`
	Address    string   `json:"address"`
	Latitude   *float64 `json:"latitude"`
	Longitude  *float64 `json:"longitude"`
	Notes      string   `json:"notes"`
	ResellerID *string  `json:"reseller_id"`
}

type updateCustomerAccessRequest struct {
	ConnectionType string  `json:"connection_type"`
	PPPoEUsername  string  `json:"pppoe_username"`
	PPPoEPassword  string  `json:"pppoe_password"`
	IPAddress      string  `json:"ip_address"`
	RouterID       *string `json:"router_id"`
	ODPPortID      *string `json:"odp_port_id"`
}

type updateCustomerServiceRequest struct {
	PackageID        *string `json:"package_id"`
	BillingProfileID *string `json:"billing_profile_id"` // nil = tidak diubah
	JoinDate         string  `json:"join_date"`
	InvoiceDate      string  `json:"invoice_date"`
	BillingDueDate   string  `json:"billing_due_date"`
	BillingType      string  `json:"billing_type"`
	CustomPrice      *int64  `json:"custom_price"`
	Discount         *int64  `json:"discount"`
	AdditionalFee    *int64  `json:"additional_fee"`
	FeeDescription   string  `json:"fee_description"`
}

func (h *CustomerHandler) NextCode(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	code, err := h.customerService.NextCode(c.Context(), tenantID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal generate kode"})
	}
	return c.JSON(fiber.Map{"code": code})
}

func (h *CustomerHandler) Create(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	var req createCustomerRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Nama wajib diisi"})
	}

	customer, err := h.customerService.Create(c.Context(), service.CreateCustomerInput{
		TenantID:         tenantID,
		Name:             req.Name,
		NIK:              req.NIK,
		Phone:            req.Phone,
		Email:            req.Email,
		Address:          req.Address,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		ConnectionType:   req.ConnectionType,
		PPPoEUsername:    req.PPPoEUsername,
		PPPoEPassword:    req.PPPoEPassword,
		IPAddress:        req.IPAddress,
		PackageID:        req.PackageID,
		RouterID:         req.RouterID,
		ODPPortID:        req.ODPPortID,
		JoinDate:         req.JoinDate,
		BillingType:      req.BillingType,
		BillingProfileID: req.BillingProfileID,
		CustomPrice:      req.CustomPrice,
		Discount:         req.Discount,
		AdditionalFee:    req.AdditionalFee,
		FeeDescription:   req.FeeDescription,
		Notes:            req.Notes,
		SerialNumber:     req.SerialNumber,
		ONTVendor:        req.ONTVendor,
		ONTModel:         req.ONTModel,
		ReferralCodeUsed: req.ReferralCodeUsed,
		ResellerID:       req.ResellerID,
	})
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(customer)
}

func (h *CustomerHandler) GetByID(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	customer, err := h.customerService.GetByID(c.Context(), tenantID, customerID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(newCustomerDetailResponse(customer))
}

func (h *CustomerHandler) Update(c *fiber.Ctx) error {
	return h.UpdateProfile(c)
}

func (h *CustomerHandler) UpdateProfile(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	var req updateCustomerProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	customer, err := h.customerService.UpdateProfile(c.Context(), tenantID, customerID, service.UpdateCustomerProfileInput{
		Name:       req.Name,
		NIK:        req.NIK,
		Phone:      req.Phone,
		Email:      req.Email,
		Address:    req.Address,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		Notes:      req.Notes,
		ResellerID: req.ResellerID,
	})
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(customer)
}

func (h *CustomerHandler) UpdateAccess(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	var req updateCustomerAccessRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	customer, err := h.customerService.UpdateAccess(c.Context(), tenantID, customerID, service.UpdateCustomerAccessInput{
		ConnectionType: req.ConnectionType,
		PPPoEUsername:  req.PPPoEUsername,
		PPPoEPassword:  req.PPPoEPassword,
		IPAddress:      req.IPAddress,
		RouterID:       req.RouterID,
		ODPPortID:      req.ODPPortID,
	})
	if err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(customer)
}

func (h *CustomerHandler) UpdateService(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	var req updateCustomerServiceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format request tidak valid"})
	}

	// Ambil customer lama untuk kalkulasi prorata dan deteksi perubahan
	oldCustomer, err := h.customerService.GetByID(c.Context(), tenantID, customerID)
	if err != nil || oldCustomer == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Pelanggan tidak ditemukan"})
	}

	var joinDate *time.Time
	if req.JoinDate != "" {
		parsed, err := parseRequestDate(req.JoinDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format join_date tidak valid"})
		}
		joinDate = &parsed
	}
	var invoiceDate *time.Time
	if req.InvoiceDate != "" {
		parsed, err := parseRequestDate(req.InvoiceDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format invoice_date tidak valid"})
		}
		invoiceDate = &parsed
	}
	var billingDueDate *time.Time
	if req.BillingDueDate != "" {
		parsed, err := parseRequestDate(req.BillingDueDate)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Format billing_due_date tidak valid"})
		}
		billingDueDate = &parsed
	}

	// 1. Tangani perubahan billing profile (harus sebelum UpdateServicePackage)
	//    ChangeBillingProfile menangani logika pending vs langsung.
	profileChanged := false
	if req.BillingProfileID != nil && h.invoiceService != nil {
		oldProfileID := ""
		if oldCustomer.BillingProfileID != nil {
			oldProfileID = *oldCustomer.BillingProfileID
		}
		newProfileID := *req.BillingProfileID
		if newProfileID != oldProfileID {
			if err := h.invoiceService.ChangeBillingProfile(c.Context(), tenantID, customerID, newProfileID); err != nil {
				log.Printf("[handler] ChangeBillingProfile customer %s: %v", customerID, err)
			} else {
				profileChanged = true
			}
		}
	}

	// 2. Update data paket & tagihan
	customer, err := h.customerService.UpdateServicePackage(c.Context(), tenantID, customerID, service.UpdateCustomerServiceInput{
		PackageID:      req.PackageID,
		JoinDate:       joinDate,
		InvoiceDate:    invoiceDate,
		BillingDueDate: billingDueDate,
		BillingType:    req.BillingType,
		CustomPrice:    req.CustomPrice,
		Discount:       req.Discount,
		AdditionalFee:  req.AdditionalFee,
		FeeDescription: req.FeeDescription,
	})
	if err != nil {
		return h.handleError(c, err)
	}

	// 3. Hitung prorata jika harga berubah dan ada invoice berjalan
	var prorataAdjustment int64
	if h.invoiceService != nil {
		oldPrice := oldCustomer.CustomPrice
		var oldPriceVal int64
		if oldPrice != nil {
			oldPriceVal = *oldPrice
		} else if oldCustomer.Package != nil {
			oldPriceVal = oldCustomer.Package.Price
		}

		newPriceVal := oldPriceVal
		if req.CustomPrice != nil {
			newPriceVal = *req.CustomPrice
		} else if req.PackageID != nil && (oldCustomer.PackageID == nil || *req.PackageID != *oldCustomer.PackageID) {
			// Paket berubah tapi tidak ada custom_price: harga baru akan diambil dari paket baru
			// Kalkulasi prorata ditangani dengan price=0 (tidak ada prorata jika harga paket tidak diketahui di sini)
			newPriceVal = 0
		}

		if oldPriceVal != newPriceVal && newPriceVal > 0 {
			adj, err := h.invoiceService.ApplyProratedPackageChange(c.Context(), tenantID, customerID, oldPriceVal, newPriceVal, time.Now())
			if err != nil {
				log.Printf("[handler] prorata customer %s: %v", customerID, err)
			} else {
				prorataAdjustment = adj
			}
		}
	}

	resp := fiber.Map{"data": customer}
	if prorataAdjustment != 0 {
		resp["prorata_adjustment"] = prorataAdjustment
		if prorataAdjustment > 0 {
			resp["prorata_info"] = "Invoice periode berjalan ditambah penyesuaian upgrade paket"
		} else {
			resp["prorata_info"] = "Invoice periode berjalan dikurangi penyesuaian downgrade paket"
		}
	}
	if profileChanged {
		resp["billing_profile_info"] = "Profil billing baru akan berlaku mulai periode berikutnya jika ada invoice aktif"
	}

	return c.JSON(resp)
}

func parseRequestDate(value string) (time.Time, error) {
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.RFC3339, value)
}

func (h *CustomerHandler) Delete(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	if err := h.customerService.Delete(c.Context(), tenantID, customerID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Pelanggan berhasil dihapus"})
}

func (h *CustomerHandler) List(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	filter := repository.CustomerFilter{
		Search:   c.Query("search"),
		Status:   c.Query("status"),
		RouterID: c.Query("router_id"),
		Page:     page,
		PerPage:  perPage,
	}

	customers, total, err := h.customerService.List(c.Context(), tenantID, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal memuat daftar pelanggan"})
	}

	return c.JSON(fiber.Map{
		"data":     customers,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	})
}

func (h *CustomerHandler) Isolate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	if err := h.customerService.Isolate(c.Context(), tenantID, customerID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Pelanggan berhasil diisolir"})
}

func (h *CustomerHandler) Activate(c *fiber.Ctx) error {
	tenantID, _ := c.Locals("tenant_id").(string)
	customerID := c.Params("id")

	if err := h.customerService.Activate(c.Context(), tenantID, customerID); err != nil {
		return h.handleError(c, err)
	}

	return c.JSON(fiber.Map{"message": "Pelanggan berhasil diaktifkan"})
}

func (h *CustomerHandler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrCustomerNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrCustomerAlreadyActive),
		errors.Is(err, service.ErrCustomerNotActive):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrCustomerCodeExists),
		errors.Is(err, service.ErrPPPoEUsernameExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrPPPoEUsernameInvalid),
		errors.Is(err, service.ErrPPPoEPasswordInvalid),
		errors.Is(err, service.ErrODPPortInvalid):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrODPPortTaken):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, service.ErrCustomerLimitReached):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": err.Error()})
	default:
		log.Printf("[CustomerHandler] unhandled error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kesalahan server internal"})
	}
}
