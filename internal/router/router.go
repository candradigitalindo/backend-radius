package router

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/handler"
	"github.com/candrasyahputra/radius-server/internal/middleware"
	"github.com/candrasyahputra/radius-server/internal/pkg/genieacs"
	"github.com/candrasyahputra/radius-server/internal/pkg/token"
	"github.com/candrasyahputra/radius-server/internal/pkg/vpn"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
)

type Dependencies struct {
	Config *config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
}

func Setup(app *fiber.App, deps *Dependencies) {
	authMiddleware := middleware.NewAuthMiddleware(deps.Config.JWT.PublicKeyPath).WithRedis(deps.Redis)

	// Token manager
	tokenManager := token.NewManager(deps.Config.JWT)

	// Repositories
	userRepo := repository.NewUserRepository(deps.DB)
	customerRepo := repository.NewCustomerRepository(deps.DB)
	sessionRepo := repository.NewSessionRepository(deps.DB)
	routerRepo := repository.NewRouterRepository(deps.DB)
	packageRepo := repository.NewPackageRepository(deps.DB)
	tenantRepo := repository.NewTenantRepository(deps.DB)
	invoiceRepo := repository.NewInvoiceRepository(deps.DB)
	paymentRepo := repository.NewPaymentRepository(deps.DB)
	ticketRepo := repository.NewTicketRepository(deps.DB)
	voucherProductRepo := repository.NewVoucherProductRepository(deps.DB)
	voucherRepo := repository.NewVoucherRepository(deps.DB)
	voucherPaymentRepo := repository.NewVoucherPaymentRepository(deps.DB)
	expenseCategoryRepo := repository.NewExpenseCategoryRepository(deps.DB)
	expenseRepo := repository.NewExpenseRepository(deps.DB)
	dashboardRepo := repository.NewDashboardRepository(deps.DB)
	broadcastRepo := repository.NewBroadcastRepository(deps.DB)
	customerLogRepo := repository.NewCustomerLogRepository(deps.DB)
	settingRepo := repository.NewSettingRepository(deps.DB)
	oltRepo := repository.NewOLTRepository(deps.DB)
	odpRepo := repository.NewODPRepository(deps.DB)
	ontRepo := repository.NewONTRepository(deps.DB)
	ftthRepo := repository.NewFTTHRepository(deps.DB)
	reminderRepo := repository.NewReminderRepository(deps.DB)
	reportRepo := repository.NewReportRepository(deps.DB)
	bandwidthRepo := repository.NewBandwidthRepository(deps.DB)
	notificationRepo := repository.NewNotificationRepository(deps.DB)
	ipamRepo := repository.NewIPAMRepository(deps.DB)
	resellerRepo := repository.NewResellerRepository(deps.DB)
	rewardRepo := repository.NewRewardRepository(deps.DB)
	subscriptionRepo := repository.NewSubscriptionRepository(deps.DB)
	roleRepo := repository.NewRoleRepository(deps.DB)
	billingProfileRepo := repository.NewBillingProfileRepository(deps.DB)
	auditLogRepo := repository.NewAuditLogRepository(deps.DB)
	waRepo := repository.NewWhatsAppRepository(deps.DB)
	waTemplateRepo := repository.NewWABroadcastTemplateRepository(deps.DB)
	adminRepo := repository.NewAdminRepository(deps.DB)

	// VPN Manager
	vpnMgr := vpn.NewManager(
		deps.Config.VPN.Interface,
		deps.Config.VPN.Subnet,
		deps.Config.VPN.ServerIP,
		deps.Config.VPN.ListenPort,
		deps.Config.VPN.PublicIP,
	)

	// Restore WireGuard peers from database
	if vpnMgr.IsAvailable() {
		dbPeers, err := routerRepo.ListVPNPeers(context.Background())
		if err != nil {
			log.Printf("[VPN] Failed to load peers from DB: %v", err)
		} else if len(dbPeers) > 0 {
			restorePeers := make([]struct{ PublicKey, VPNIP string }, len(dbPeers))
			for i, p := range dbPeers {
				restorePeers[i] = struct{ PublicKey, VPNIP string }{p.PublicKey, p.VPNIP}
			}
			vpnMgr.RestorePeers(restorePeers)
		}
	}

	// WhatsApp client
	waClient := whatsapp.NewClient(deps.Config.WhatsApp)

	// Services
	tenantService := service.NewTenantService(tenantRepo, userRepo, roleRepo).WithWAClient(waClient).WithBaseURL(deps.Config.App.URL).WithReminderRepo(reminderRepo)
	authService := service.NewAuthService(userRepo, customerRepo, tenantRepo, roleRepo, tokenManager, deps.Redis).WithTenantService(tenantService).WithSubscriptionRepo(subscriptionRepo).WithReminderRepo(reminderRepo)
	userService := service.NewUserService(userRepo, roleRepo)
	roleService := service.NewRoleService(roleRepo, userRepo)
	customerService := service.NewCustomerService(customerRepo, sessionRepo)
	snmpService := service.NewSNMPService(oltRepo, vpnMgr)
	routerService := service.NewRouterService(routerRepo, sessionRepo, vpnMgr).WithSNMP(snmpService).
		WithAppConfig(deps.Config.App.URL, deps.Config.RADIUS.Secret).
		WithIfaceRepo(repository.NewRouterIfaceRepository(deps.DB))
	packageService := service.NewPackageService(packageRepo)
	invoiceService := service.NewInvoiceService(invoiceRepo, paymentRepo, customerRepo).WithTenantRepo(tenantRepo).WithWAClient(waClient).WithBaseURL(deps.Config.App.URL).WithSettingRepo(settingRepo).WithBillingProfileRepo(billingProfileRepo)
	ticketService := service.NewTicketService(ticketRepo)
	mobileService := service.NewMobileService(customerRepo, invoiceRepo, ticketRepo, packageRepo, tokenManager)
	voucherService := service.NewVoucherService(voucherProductRepo, voucherRepo, voucherPaymentRepo).WithTenantRepo(tenantRepo).WithWAClient(waClient).WithBaseURL(deps.Config.App.URL)
	expenseService := service.NewExpenseService(expenseCategoryRepo, expenseRepo)
	dashboardService := service.NewDashboardService(dashboardRepo)
	broadcastService := service.NewBroadcastService(broadcastRepo, customerRepo, waClient)
	broadcastService.StartScheduler()
	customerLogService := service.NewCustomerLogService(customerLogRepo)
	settingService := service.NewSettingService(settingRepo)
	oltService := service.NewOLTService(oltRepo).WithVPNManager(vpnMgr)
	go oltService.RestoreAllRoutes(context.Background())
	odpService := service.NewODPService(odpRepo, customerRepo, ontRepo)
	ontService := service.NewONTService(ontRepo).WithCustomerLog(customerLogRepo)
	ftthService := service.NewFTTHService(ftthRepo)
	reminderService := service.NewReminderService(reminderRepo, invoiceRepo, customerRepo, tenantRepo, waClient).WithSettingRepo(settingRepo).WithPaymentRepo(paymentRepo).WithBaseURL(deps.Config.App.URL)
	reportService := service.NewReportService(reportRepo)
	exportService := service.NewExportService(reportRepo)
	bandwidthService := service.NewBandwidthService(bandwidthRepo)
	notificationService := service.NewNotificationService(notificationRepo, customerRepo, deps.Config.FCM.ProjectID, deps.Config.FCM.CredentialsFile, deps.Config.FCM.Enabled).WithWAClient(waClient)
	// Enable web-push + in-app notifications for invoice/payment events.
	invoiceService.WithNotificationService(notificationService)
	ipamService := service.NewIPAMService(ipamRepo)
	resellerService := service.NewResellerService(resellerRepo)
	rewardService := service.NewRewardService(rewardRepo).WithInvoice(invoiceRepo)
	subscriptionService := service.NewSubscriptionService(subscriptionRepo, tenantRepo).WithPG(&deps.Config.PG, deps.Config.App.URL).WithReminderRepo(reminderRepo).WithWAClient(waClient).WithSettingRepo(settingRepo)
	adminService := service.NewAdminService(adminRepo).WithTenantService(tenantService)
	billingProfileService := service.NewBillingProfileService(billingProfileRepo)

	// GenieACS TR-069 client & service
	genieacsClient := genieacs.NewClient(deps.Config.GenieACS)
	genieacsService := service.NewGenieACSService(genieacsClient, ontRepo, customerRepo).
		WithWAClient(waClient).
		WithSettingRepo(settingRepo)

	// Link GenieACS to customer service for FTTH auto-provisioning
	customerService.WithGenieACS(ontRepo, genieacsService)
	customerService.WithCWMPPort(deps.Config.App.CWMPPort)
	customerService.WithCWMPURL(deps.Config.App.CWMPURL)
	customerService.WithInvoice(invoiceRepo)
	customerService.WithBandwidth(bandwidthRepo)
	customerService.WithReward(rewardService)
	customerService.WithTenant(tenantRepo, deps.Config.App.URL)
	invoiceService.WithReward(rewardService)
	invoiceService.WithReseller(resellerService)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	authHandler.WithResetDeps(userRepo, tenantRepo, deps.Redis, waClient)
	authHandler.WithReminderRepo(reminderRepo)
	userHandler := handler.NewUserHandler(userService)
	roleHandler := handler.NewRoleHandler(roleService)
	customerHandler := handler.NewCustomerHandler(customerService).WithInvoiceService(invoiceService)
	routerHandler := handler.NewRouterHandler(routerService)
	internalHandler := handler.NewInternalHandler(tenantRepo, customerRepo, deps.Config.WhatsApp.APISecret)
	packageHandler := handler.NewPackageHandler(packageService)
	baseURL := deps.Config.App.URL
	tenantHandler := handler.NewTenantHandler(tenantService).WithWebhookBaseURL(baseURL)
	invoiceHandler := handler.NewInvoiceHandler(invoiceService)
	ticketHandler := handler.NewTicketHandler(ticketService)
	voucherHandler := handler.NewVoucherHandler(voucherService)
	expenseHandler := handler.NewExpenseHandler(expenseService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	broadcastHandler := handler.NewBroadcastHandler(broadcastService).WithBaseURL(deps.Config.App.URL)
	customerLogHandler := handler.NewCustomerLogHandler(customerLogService)
	settingHandler := handler.NewSettingHandler(settingService)
	oltHandler := handler.NewOLTHandler(oltService)
	odpHandler := handler.NewODPHandler(odpService)
	ontHandler := handler.NewONTHandler(ontService)
	genieacsHandler := handler.NewGenieACSHandler(genieacsService)
	ftthHandler := handler.NewFTTHHandler(ftthService)
	reminderHandler := handler.NewReminderHandler(reminderService)
	reportHandler := handler.NewReportHandler(reportService)
	exportHandler := handler.NewExportHandler(reportService, exportService)
	bandwidthHandler := handler.NewBandwidthHandler(bandwidthService, sessionRepo)
	notificationHandler := handler.NewNotificationHandler(notificationService).WithBaseURL(deps.Config.App.URL)
	ipamHandler := handler.NewIPAMHandler(ipamService)
	resellerHandler := handler.NewResellerHandler(resellerService)
	rewardHandler := handler.NewRewardHandler(rewardService)
	webhookHandler := handler.NewWebhookHandler(invoiceService).
		WithVoucherService(voucherService).
		WithSubscriptionService(subscriptionService)
	publicPaymentHandler := handler.NewPublicPaymentHandler(invoiceService)
	publicVoucherHandler := handler.NewPublicVoucherHandler(voucherService, tenantService)
	isolirPageHandler := handler.NewIsolirPageHandler(settingService, tenantService, customerService)
	mobileHandler := handler.NewMobileHandler(mobileService)
	themeHandler := handler.NewThemeHandler(settingService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	snmpHandler := handler.NewSNMPHandler(snmpService).WithOLTService(oltService)
	waHandler := handler.NewWhatsAppHandler(waClient, waRepo).WithBaseURL(deps.Config.App.URL)
	waTemplateHandler := handler.NewWATemplateHandler(waTemplateRepo)
	auditLogHandler := handler.NewAuditLogHandler(auditLogRepo)
	adminHandler := handler.NewAdminHandler(adminService).WithSubscriptionService(subscriptionService).WithReminderRepo(reminderRepo)
	saSettingsHandler := handler.NewSASettingsHandler(tenantService, settingService, waClient).WithBaseURL(deps.Config.App.URL)
	billingProfileHandler := handler.NewBillingProfileHandler(billingProfileService)
	wsHandler := handler.NewWSHandler(dashboardService, bandwidthService, routerService, deps.Config.JWT.PublicKeyPath)
	i18nHandler := handler.NewI18nHandler()
	_ = userHandler
	_ = roleHandler
	_ = broadcastHandler
	_ = customerLogHandler
	_ = settingHandler
	_ = oltHandler
	_ = odpHandler
	_ = ftthHandler
	_ = reminderHandler
	_ = reportHandler
	_ = bandwidthHandler
	_ = notificationHandler
	_ = ipamHandler
	_ = themeHandler
	_ = snmpHandler

	// API V1 Group
	v1 := app.Group("/api/v1")
	v1.Use("/ws", wsHandler.UpgradeCheck)
	v1.Get("/ws", wsHandler.Handle())

	// Public static files (for temporary WhatsApp uploads)
	app.Static("/temp", "./storage/temp")

	// Limiter for public endpoints. Keyed by real client IP (Fiber now reads
	// X-Forwarded-For via the trusted proxy, so this is no longer a global key).
	// 30/min/IP gives headroom for login+refresh and for several staff behind a
	// shared office NAT, while staying low enough to deter password brute-force.
	publicLimiter := limiter.New(limiter.Config{
		Max:        30,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	// Limiter for customer portal endpoints. Customers of one ISP usually share
	// a single public IP (behind the ISP's NAT gateway), so a per-IP limit of 10
	// is far too low and trips 429 as soon as a few customers use the portal.
	// Scope the key per IP+tenant and allow a much higher ceiling.
	portalLimiter := limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP() + "|" + c.Params("slug")
		},
	})

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register/send-otp", publicLimiter, authHandler.SendRegistrationOTP)
	auth.Post("/register", publicLimiter, authHandler.Register)
	auth.Get("/plans", publicLimiter, authHandler.ListPlans)
	auth.Post("/select-plan", authMiddleware.Handle(), authHandler.SelectPlan)
	auth.Post("/login", publicLimiter, authHandler.Login)
	auth.Post("/refresh", publicLimiter, authHandler.RefreshToken)
	auth.Post("/reset-pin", publicLimiter, authHandler.RequestResetPIN)
	auth.Post("/reset-password", publicLimiter, authHandler.ResetPasswordWithPIN)

	// Router heartbeat (public - called by MikroTik)
	// Internal m2m endpoint for the n8n CS bot (guarded by WA_API_SECRET, not JWT).
	v1.Get("/internal/tenant-by-phone", publicLimiter, internalHandler.TenantByPhone)

	v1.Post("/routers/heartbeat", publicLimiter, routerHandler.Heartbeat)
	v1.Post("/routers/interface-stats", publicLimiter, routerHandler.PushInterfaceStats)

	// Swagger UI at root (port 3000)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("./static/swagger.html")
	})
	app.Get("/swagger.json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		return c.SendFile("./docs/swagger.json")
	})

	// CWMP TR-069 proxy (public - called by ONT devices)
	cwmpHandler := handler.NewCWMPHandler(genieacsClient, tenantRepo, customerRepo, ontRepo, deps.Config.GenieACS.CWMPURL)
	app.All("/cwmp", cwmpHandler.Handle)

	// Payment gateway webhooks (per-tenant invoice & voucher; global subscription)
	webhooks := v1.Group("/webhooks")
	webhooks.Post("/tripay/:slug", publicLimiter, webhookHandler.TripayCallback)
	webhooks.Post("/midtrans/:slug", publicLimiter, webhookHandler.MidtransCallback)
	webhooks.Post("/xendit/:slug", publicLimiter, webhookHandler.XenditCallback)
	webhooks.Post("/tripay/:slug/voucher", publicLimiter, webhookHandler.TripayVoucherCallback)
	webhooks.Post("/midtrans/:slug/voucher", publicLimiter, webhookHandler.MidtransVoucherCallback)
	webhooks.Post("/xendit/:slug/voucher", publicLimiter, webhookHandler.XenditVoucherCallback)
	webhooks.Post("/subscription/tripay", publicLimiter, webhookHandler.TripaySubscriptionCallback)
	webhooks.Post("/subscription/midtrans", publicLimiter, webhookHandler.MidtransSubscriptionCallback)
	webhooks.Post("/subscription/xendit", publicLimiter, webhookHandler.XenditSubscriptionCallback)

	// Public payment page
	publicPay := v1.Group("/public/pay")
	publicPay.Get("/:tenant_id/:customer_code", publicLimiter, publicPaymentHandler.GetCustomerInfo)
	publicPay.Post("/:tenant_id/:customer_code", publicLimiter, publicPaymentHandler.CreatePayment)
	publicPay.Get("/check/:trx_id", publicLimiter, publicPaymentHandler.CheckPaymentStatus)

	// Public isolir page
	app.Get("/isolir/:tenant_slug/:customer_code", isolirPageHandler.GetPage)

	// Public portal login
	portalHandler := handler.NewPortalHandler(customerRepo, invoiceRepo, ticketRepo, paymentRepo, tenantRepo, tokenManager)
	portalHandler.WithResetDeps(deps.Redis, waClient)
	portalHandler.WithReminderRepo(reminderRepo)
	portalHandler.WithDeviceService(ontRepo, genieacsService)
	portalHandler.WithPackageRepo(packageRepo)
	portalHandler.WithRewardRepo(rewardRepo)
	portalHandler.WithInvoiceService(invoiceService)
	publicPortal := v1.Group("/public/portal")
	publicPortal.Get("/:slug", portalLimiter, portalHandler.GetTenantInfo)
	publicPortal.Post("/:slug/login", portalLimiter, portalHandler.PortalLogin)
	publicPortal.Post("/:slug/reset-pin", portalLimiter, portalHandler.RequestResetPIN)
	publicPortal.Post("/:slug/reset-password", portalLimiter, portalHandler.ResetPasswordWithPIN)

	// Public voucher store
	publicStore := v1.Group("/public/store")
	publicStore.Get("/:tenant_slug", publicLimiter, publicVoucherHandler.GetStoreProducts)
	publicStore.Post("/:tenant_slug/buy", publicLimiter, publicVoucherHandler.PurchaseVoucher)

	// Mobile App API — public auth
	v1.Post("/mobile/auth/login", publicLimiter, mobileHandler.Login)

	// Language middleware
	v1.Use(middleware.LangMiddleware())

	// i18n routes (public)
	v1.Get("/languages", i18nHandler.GetLanguages)
	v1.Get("/translations", i18nHandler.GetTranslations)

	// Auth routes (protected)
	auth.Use(authMiddleware.Handle())
	auth.Get("/me", authHandler.Me)
	auth.Put("/me", authHandler.UpdateProfile)
	auth.Put("/change-password", authHandler.ChangePassword)
	auth.Post("/logout", authHandler.Logout)

	// Portal routes (customer web portal)
	portal := v1.Group("/portal", authMiddleware.Handle(), middleware.TenantGuard())
	portal.Get("/profile", portalHandler.GetProfile)
	portal.Get("/customer", portalHandler.GetCustomer)
	portal.Get("/invoices", portalHandler.ListInvoices)
	portal.Get("/invoices/:id", portalHandler.GetInvoice)
	portal.Get("/invoices/:id/payments", portalHandler.GetInvoicePayments)
	portal.Get("/tickets", portalHandler.ListTickets)
	portal.Post("/tickets", portalHandler.CreateTicket)
	portal.Get("/tickets/:id", portalHandler.GetTicket)
	portal.Get("/tickets/:id/messages", portalHandler.GetTicketMessages)
	portal.Post("/tickets/:id/messages", portalHandler.ReplyTicket)
	portal.Put("/change-password", portalHandler.ChangePassword)
	portal.Get("/device", portalHandler.GetDevice)
	portal.Put("/device/wifi", portalHandler.SetDeviceWiFi)
	portal.Post("/device/reboot", portalHandler.RebootDevice)
	// Web push (PWA) device token registration
	portal.Post("/push/register", notificationHandler.RegisterDevice)
	portal.Post("/push/unregister", notificationHandler.UnregisterDevice)
	portal.Get("/packages", portalHandler.GetAvailablePackages)
	portal.Post("/change-package", portalHandler.RequestPackageChange)
	portal.Get("/referral", portalHandler.GetReferralInfo)
	portal.Get("/payment-config", portalHandler.GetPaymentConfig)
	portal.Post("/invoices/:id/pay-gateway", portalHandler.PayInvoiceGateway)

	// Protected routes (staff only)
	protected := v1.Group("/", authMiddleware.Handle(), middleware.TenantGuard(), middleware.StaffGuard(), middleware.NewPermissionLoader(roleRepo), middleware.SubscriptionGuard(tenantRepo))

	// Billing Profile routes
	billingProfiles := protected.Group("/billing-profiles")
	billingProfiles.Get("/", billingProfileHandler.List)
	billingProfiles.Post("/", billingProfileHandler.Create)
	billingProfiles.Get("/:id", billingProfileHandler.GetByID)
	billingProfiles.Put("/:id", billingProfileHandler.Update)
	billingProfiles.Delete("/:id", billingProfileHandler.Delete)
	billingProfiles.Put("/:id/default", billingProfileHandler.SetDefault)

	// Current tenant routes
	protected.Get("/tenant", tenantHandler.GetCurrent)
	protected.Put("/tenant", tenantHandler.Update)
	protected.Put("/tenant/settings", tenantHandler.UpdateSettings)
	protected.Post("/tenant/settings/test", tenantHandler.TestPGConnection)
	protected.Get("/tenant/webhook-urls", tenantHandler.GetWebhookURLs)

	// Tenant subscription routes
	subscription := protected.Group("/subscription")
	subscription.Get("/plans", subscriptionHandler.ListPlans)
	subscription.Get("/plans/:id", subscriptionHandler.GetPlan)
	subscription.Post("/subscribe", subscriptionHandler.Subscribe)
	subscription.Get("/orders", subscriptionHandler.ListOrders)
	subscription.Get("/orders/:id", subscriptionHandler.GetOrder)
	subscription.Post("/orders/:id/pay", subscriptionHandler.CreatePayment)
	subscription.Post("/orders/:id/confirm", middleware.RoleGuard("superadmin"), subscriptionHandler.ConfirmPayment)
	subscription.Get("/status", subscriptionHandler.GetStatus)

	// User routes
	users := protected.Group("/users", middleware.PermissionGuard("users.view"))
	users.Get("/", userHandler.List)
	users.Post("/", middleware.PermissionGuard("users.create"), userHandler.Create)
	users.Put("/:id", middleware.PermissionGuard("users.edit"), userHandler.Update)
	users.Delete("/:id", middleware.PermissionGuard("users.delete"), userHandler.Delete)
	users.Put("/:id/toggle-active", middleware.PermissionGuard("users.edit"), userHandler.ToggleActive)

	// Role routes
	roles := protected.Group("/roles", middleware.PermissionGuard("roles.view"))
	roles.Get("/", roleHandler.List)
	roles.Get("/permissions", roleHandler.ListPermissions)
	roles.Post("/", middleware.PermissionGuard("roles.create"), roleHandler.Create)
	roles.Put("/:id", middleware.PermissionGuard("roles.edit"), roleHandler.Update)
	roles.Delete("/:id", middleware.PermissionGuard("roles.delete"), roleHandler.Delete)

	// Broadcast routes
	broadcasts := protected.Group("/broadcasts", middleware.PermissionGuard("broadcasts.view"))
	broadcasts.Get("/", broadcastHandler.List)
	broadcasts.Post("/", middleware.PermissionGuard("broadcasts.create"), broadcastHandler.Send)
	broadcasts.Get("/:id", broadcastHandler.GetByID)
	broadcasts.Delete("/:id", middleware.PermissionGuard("broadcasts.delete"), broadcastHandler.Delete)

	// Settings routes
	settings := protected.Group("/settings", middleware.PermissionGuard("settings.view"))
	settings.Get("/", settingHandler.List)
	settings.Get("/theme", themeHandler.GetTenantTheme)
	settings.Get("/:key", settingHandler.Get)
	settings.Post("/", middleware.PermissionGuard("settings.edit"), settingHandler.Set)
	settings.Put("/bulk", middleware.PermissionGuard("settings.edit"), settingHandler.BulkSet)
	settings.Put("/theme", middleware.PermissionGuard("settings.edit"), themeHandler.UpdateTenantTheme)
	settings.Delete("/:key", middleware.PermissionGuard("settings.edit"), settingHandler.Delete)
	protected.Get("/user/preferences", themeHandler.GetPreferences)
	protected.Put("/user/preferences", themeHandler.UpdatePreferences)

	// Reminder routes
	reminders := protected.Group("/reminders", middleware.PermissionGuard("reminders.view"))
	reminders.Get("/", reminderHandler.List)
	reminders.Post("/", middleware.PermissionGuard("reminders.create"), reminderHandler.Create)
	reminders.Get("/:id", reminderHandler.GetByID)
	reminders.Put("/:id", middleware.PermissionGuard("reminders.edit"), reminderHandler.Update)
	reminders.Delete("/:id", middleware.PermissionGuard("reminders.delete"), reminderHandler.Delete)
	reminders.Post("/trigger", middleware.PermissionGuard("reminders.edit"), reminderHandler.Trigger)

	// Admin notification routes
	notifications := protected.Group("/notifications", middleware.PermissionGuard("notifications.view"))
	notifications.Get("/", notificationHandler.AdminList)
	notifications.Put("/:id/read", notificationHandler.MarkRead)
	notifications.Post("/send", middleware.PermissionGuard("notifications.send"), notificationHandler.SendNotification)
	notifications.Post("/broadcast", middleware.PermissionGuard("notifications.send"), notificationHandler.BroadcastNotification)

	// Customer routes
	customers := protected.Group("/customers", middleware.PermissionGuard("customers.view"))
	customers.Get("/next-code", customerHandler.NextCode)
	customers.Get("/", customerHandler.List)
	customers.Post("/", middleware.PermissionGuard("customers.create"), customerHandler.Create)
	customers.Get("/:id", customerHandler.GetByID)
	customers.Get("/:id/logs", customerLogHandler.ListByCustomer)
	customers.Get("/:id/ont", ontHandler.GetByCustomer)
	customers.Put("/:id", middleware.PermissionGuard("customers.edit"), customerHandler.Update)
	customers.Put("/:id/profile", middleware.PermissionGuard("customers.edit"), customerHandler.UpdateProfile)
	customers.Put("/:id/access", middleware.PermissionGuard("customers.edit"), customerHandler.UpdateAccess)
	customers.Put("/:id/service", middleware.PermissionGuard("customers.edit"), customerHandler.UpdateService)
	customers.Delete("/:id", middleware.PermissionGuard("customers.delete"), customerHandler.Delete)
	customers.Post("/:id/isolate", middleware.PermissionGuard("customers.isolate"), customerHandler.Isolate)
	customers.Post("/:id/activate", middleware.PermissionGuard("customers.isolate"), customerHandler.Activate)

	// ONT routes
	onts := protected.Group("/onts", middleware.PermissionGuard("onts.view"))
	onts.Get("/", ontHandler.List)
	onts.Post("/", middleware.PermissionGuard("onts.create"), ontHandler.Create)
	onts.Get("/:id", ontHandler.GetByID)
	onts.Put("/:id", middleware.PermissionGuard("onts.edit"), ontHandler.Update)
	onts.Delete("/:id", middleware.PermissionGuard("onts.delete"), ontHandler.Delete)
	onts.Post("/:id/sync", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.SyncONT)
	onts.Post("/:id/reboot", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.RebootDevice)
	onts.Post("/:id/wifi", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.SetWiFi)
	onts.Post("/:id/provision", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.ProvisionONT)
	onts.Get("/:id/diagnostics", middleware.PermissionGuard("genieacs.view"), genieacsHandler.GetDiagnostics)
	onts.Post("/:id/pppoe", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.SetPPPoE)

	// GenieACS device routes
	genieacs := protected.Group("/genieacs", middleware.PermissionGuard("genieacs.view"))
	genieacs.Get("/devices", genieacsHandler.ListDevices)
	genieacs.Get("/devices/:id", genieacsHandler.GetDevice)
	genieacs.Get("/tasks", genieacsHandler.ListTasks)
	genieacs.Get("/ont-dashboard", genieacsHandler.ONTDashboard)
	genieacs.Get("/devices/serial/:serial", genieacsHandler.GetDeviceBySerial)
	genieacs.Get("/devices/serial/:serial/status", genieacsHandler.GetDeviceStatus)
	genieacs.Post("/devices/serial/:serial/reboot", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.RebootBySerial)
	genieacs.Post("/devices/serial/:serial/wifi", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.SetWiFiBySerial)
	genieacs.Post("/bulk-provision", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.BulkProvision)
	genieacs.Post("/discover", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.DiscoverONTs)
	genieacs.Post("/auto-match", middleware.PermissionGuard("genieacs.manage"), genieacsHandler.AutoMatchONTs)

	// Router routes
	routers := protected.Group("/routers", middleware.PermissionGuard("routers.view"))
	routers.Get("/", routerHandler.List)
	routers.Post("/", middleware.PermissionGuard("routers.create"), routerHandler.Create)
	routers.Get("/:id", routerHandler.GetByID)
	routers.Put("/:id", middleware.PermissionGuard("routers.edit"), routerHandler.Update)
	routers.Delete("/:id", middleware.PermissionGuard("routers.delete"), routerHandler.Delete)
	routers.Post("/:id/regenerate-token", middleware.PermissionGuard("routers.edit"), routerHandler.RegenerateToken)
	routers.Get("/:id/sessions", routerHandler.ListSessions)
	routers.Get("/:id/traffic", routerHandler.GetTraffic)
	routers.Get("/:id/interfaces", routerHandler.GetInterfaces)
	routers.Get("/:id/interface-stats", routerHandler.GetInterfaceStats)
	routers.Post("/:id/test", middleware.PermissionGuard("routers.edit"), routerHandler.TestConnection)
	routers.Post("/:id/sync", middleware.PermissionGuard("routers.edit"), routerHandler.SyncSessions)
	routers.Post("/:id/vpn-key", middleware.PermissionGuard("routers.edit"), routerHandler.RegisterVPNKey)
	routers.Get("/:id/mikrotik-config", routerHandler.GetMikroTikConfig)
	routers.Get("/:id/connection-logs", routerHandler.ListConnectionLogs)
	routers.Get("/:id/ip-pool", ipamHandler.GetPoolByRouter)
	routers.Get("/vpn/peers", routerHandler.ListVPNPeers)

	// OLT routes
	olts := protected.Group("/olts", middleware.PermissionGuard("olts.view"))
	olts.Get("/", oltHandler.List)
	olts.Post("/", middleware.PermissionGuard("olts.create"), oltHandler.Create)
	olts.Get("/:id", oltHandler.GetByID)
	olts.Put("/:id", middleware.PermissionGuard("olts.edit"), oltHandler.Update)
	olts.Delete("/:id", middleware.PermissionGuard("olts.delete"), oltHandler.Delete)
	olts.Get("/:id/snmp/monitor", snmpHandler.MonitorOLT)
	olts.Post("/:id/sync", middleware.PermissionGuard("olts.edit"), snmpHandler.SyncOLT)
	olts.Get("/:id/pon-ports", oltHandler.ListPONPorts)
	olts.Post("/:id/pon-ports", middleware.PermissionGuard("olts.edit"), oltHandler.CreatePONPort)
	olts.Put("/:id/pon-ports/:portId", middleware.PermissionGuard("olts.edit"), oltHandler.UpdatePONPort)
	olts.Delete("/:id/pon-ports/:portId", middleware.PermissionGuard("olts.delete"), oltHandler.DeletePONPort)

	// ODP routes
	odps := protected.Group("/odps", middleware.PermissionGuard("odps.view"))
	odps.Get("/", odpHandler.List)
	odps.Post("/", middleware.PermissionGuard("odps.create"), odpHandler.Create)
	odps.Get("/:id", odpHandler.GetByID)
	odps.Put("/:id", middleware.PermissionGuard("odps.edit"), odpHandler.Update)
	odps.Delete("/:id", middleware.PermissionGuard("odps.delete"), odpHandler.Delete)
	odps.Get("/:id/ports", odpHandler.ListPorts)
	odps.Post("/:id/ports", middleware.PermissionGuard("odps.edit"), odpHandler.CreatePort)
	odps.Put("/:id/ports/:portId", middleware.PermissionGuard("odps.edit"), odpHandler.UpdatePort)
	odps.Delete("/:id/ports/:portId", middleware.PermissionGuard("odps.delete"), odpHandler.DeletePort)

	// Splitter routes
	splitters := protected.Group("/splitters", middleware.PermissionGuard("odps.view"))
	splitters.Get("/", odpHandler.ListSplitters)
	splitters.Post("/", middleware.PermissionGuard("odps.create"), odpHandler.CreateSplitter)
	splitters.Get("/:id", odpHandler.GetSplitter)
	splitters.Put("/:id", middleware.PermissionGuard("odps.edit"), odpHandler.UpdateSplitter)
	splitters.Delete("/:id", middleware.PermissionGuard("odps.delete"), odpHandler.DeleteSplitter)

	// FTTH routes
	ftth := protected.Group("/ftth", middleware.PermissionGuard("ftth.view"))
	ftth.Get("/stats", ftthHandler.GetStats)
	ftth.Get("/map", ftthHandler.GetMap)

	// Bandwidth routes
	bandwidth := protected.Group("/bandwidth", middleware.PermissionGuard("bandwidth.view"))
	bandwidth.Get("/summary", bandwidthHandler.GetUsageSummary)
	bandwidth.Get("/top-users", bandwidthHandler.GetTopUsers)
	bandwidth.Get("/saturation", bandwidthHandler.GetSaturationReport)
	bandwidth.Get("/customers/:id", bandwidthHandler.GetCustomerUsage)
	bandwidth.Get("/customers/:id/history", bandwidthHandler.GetCustomerUsageHistory)
	bandwidth.Get("/customers/:id/connections", bandwidthHandler.GetCustomerConnections)

	// IPAM routes
	ipPools := protected.Group("/ip-pools", middleware.PermissionGuard("ip_pools.view"))
	ipPools.Get("/", ipamHandler.ListPools)
	ipPools.Post("/", middleware.PermissionGuard("ip_pools.create"), ipamHandler.CreatePool)
	ipPools.Get("/:id", ipamHandler.GetPool)
	ipPools.Put("/:id", middleware.PermissionGuard("ip_pools.edit"), ipamHandler.UpdatePool)
	ipPools.Delete("/:id", middleware.PermissionGuard("ip_pools.delete"), ipamHandler.DeletePool)
	ipPools.Get("/:poolId/addresses", ipamHandler.ListAddresses)
	ipPools.Post("/:poolId/addresses", middleware.PermissionGuard("ip_pools.create"), ipamHandler.CreateAddress)
	ipPools.Post("/:poolId/addresses/batch", middleware.PermissionGuard("ip_pools.create"), ipamHandler.CreateAddressBatch)
	ipPools.Get("/:poolId/available", ipamHandler.FindAvailable)
	ipPools.Get("/:poolId/stats", ipamHandler.GetPoolStats)
	ipPools.Post("/:poolId/addresses/:addrId/assign", middleware.PermissionGuard("ip_pools.edit"), ipamHandler.AssignAddress)
	ipPools.Post("/:poolId/addresses/:addrId/release", middleware.PermissionGuard("ip_pools.edit"), ipamHandler.ReleaseAddress)

	// Package routes
	packages := protected.Group("/packages", middleware.PermissionGuard("packages.view"))
	packages.Get("/", packageHandler.List)
	packages.Post("/", middleware.PermissionGuard("packages.create"), packageHandler.Create)
	packages.Get("/:id", packageHandler.GetByID)
	packages.Put("/:id", middleware.PermissionGuard("packages.edit"), packageHandler.Update)
	packages.Delete("/:id", middleware.PermissionGuard("packages.delete"), packageHandler.Delete)

	// Invoice routes
	invoices := protected.Group("/invoices", middleware.PermissionGuard("invoices.view"))
	invoices.Get("/", invoiceHandler.List)
	invoices.Post("/", middleware.PermissionGuard("invoices.create"), invoiceHandler.Create)
	invoices.Post("/generate", middleware.PermissionGuard("invoices.create"), invoiceHandler.Generate)
	invoices.Get("/:id", invoiceHandler.GetByID)
	invoices.Put("/:id", middleware.PermissionGuard("invoices.edit"), invoiceHandler.Update)
	invoices.Delete("/:id", middleware.PermissionGuard("invoices.delete"), invoiceHandler.Delete)
	invoices.Post("/:id/pay", middleware.PermissionGuard("invoices.pay"), invoiceHandler.RecordPayment)
	invoices.Post("/:id/pay-gateway", middleware.PermissionGuard("invoices.pay"), invoiceHandler.CreateGatewayPayment)
	invoices.Get("/:id/payments", invoiceHandler.GetPayments)
	invoices.Post("/:id/notify", middleware.PermissionGuard("invoices.view"), invoiceHandler.Notify)

	// Payment routes
	protected.Get("/payments", invoiceHandler.ListPayments)

	// Customer log routes
	customerLogs := protected.Group("/customer-logs", middleware.PermissionGuard("customers.view"))
	customerLogs.Get("/", customerLogHandler.List)
	customerLogs.Post("/", middleware.PermissionGuard("customers.edit"), customerLogHandler.Create)
	customerLogs.Get("/:id", customerLogHandler.GetByID)

	// Expense category routes
	expenseCategories := protected.Group("/expense-categories", middleware.PermissionGuard("expenses.view"))
	expenseCategories.Get("/", expenseHandler.ListCategories)
	expenseCategories.Post("/", middleware.PermissionGuard("expenses.create"), expenseHandler.CreateCategory)
	expenseCategories.Get("/:id", expenseHandler.GetCategory)
	expenseCategories.Put("/:id", middleware.PermissionGuard("expenses.edit"), expenseHandler.UpdateCategory)
	expenseCategories.Delete("/:id", middleware.PermissionGuard("expenses.delete"), expenseHandler.DeleteCategory)

	// Expense routes
	expenses := protected.Group("/expenses", middleware.PermissionGuard("expenses.view"))
	expenses.Get("/", expenseHandler.List)
	expenses.Get("/summary", expenseHandler.Summary)
	expenses.Post("/", middleware.PermissionGuard("expenses.create"), expenseHandler.Create)
	expenses.Get("/:id", expenseHandler.GetByID)
	expenses.Put("/:id", middleware.PermissionGuard("expenses.edit"), expenseHandler.Update)
	expenses.Delete("/:id", middleware.PermissionGuard("expenses.delete"), expenseHandler.Delete)

	// Ticket routes
	tickets := protected.Group("/tickets", middleware.PermissionGuard("tickets.view"))
	tickets.Get("/", ticketHandler.List)
	tickets.Post("/", middleware.PermissionGuard("tickets.create"), ticketHandler.Create)
	tickets.Get("/:id", ticketHandler.GetByID)
	tickets.Put("/:id", middleware.PermissionGuard("tickets.edit"), ticketHandler.Update)
	tickets.Delete("/:id", middleware.PermissionGuard("tickets.delete"), ticketHandler.Delete)
	tickets.Put("/:id/status", middleware.PermissionGuard("tickets.edit"), ticketHandler.UpdateStatus)
	tickets.Put("/:id/assign", middleware.PermissionGuard("tickets.assign"), ticketHandler.Assign)
	tickets.Get("/:id/messages", ticketHandler.GetMessages)
	tickets.Post("/:id/messages", middleware.PermissionGuard("tickets.edit"), ticketHandler.AddMessage)

	// Voucher routes
	voucherProducts := protected.Group("/voucher-products", middleware.PermissionGuard("vouchers.view"))
	voucherProducts.Get("/", voucherHandler.ListProducts)
	voucherProducts.Post("/", middleware.PermissionGuard("vouchers.create"), voucherHandler.CreateProduct)
	voucherProducts.Get("/:id", voucherHandler.GetProduct)
	voucherProducts.Put("/:id", middleware.PermissionGuard("vouchers.edit"), voucherHandler.UpdateProduct)
	voucherProducts.Delete("/:id", middleware.PermissionGuard("vouchers.delete"), voucherHandler.DeleteProduct)

	vouchers := protected.Group("/vouchers", middleware.PermissionGuard("vouchers.view"))
	vouchers.Get("/", voucherHandler.ListVouchers)
	vouchers.Post("/generate", middleware.PermissionGuard("vouchers.create"), voucherHandler.Generate)
	vouchers.Get("/:id", voucherHandler.GetVoucher)
	vouchers.Delete("/:id", middleware.PermissionGuard("vouchers.delete"), voucherHandler.DeleteVoucher)

	// WhatsApp routes
	wa := protected.Group("/whatsapp", middleware.PermissionGuard("whatsapp.view"))
	wa.Get("/logs", waHandler.GetLogs)
	wa.Post("/sessions/start", middleware.PermissionGuard("whatsapp.configure"), waHandler.StartSession)
	wa.Get("/sessions/status", waHandler.GetStatus)
	wa.Get("/sessions/qr", waHandler.GetQR)
	wa.Delete("/sessions", middleware.PermissionGuard("whatsapp.configure"), waHandler.StopSession)
	wa.Post("/send", middleware.PermissionGuard("whatsapp.send"), waHandler.SendMessage)

	// WhatsApp broadcast templates
	waTemplates := wa.Group("/templates")
	waTemplates.Get("/", waTemplateHandler.List)
	waTemplates.Get("/:id", waTemplateHandler.GetByID)
	waTemplates.Post("/", waTemplateHandler.Create)
	waTemplates.Put("/:id", waTemplateHandler.Update)
	waTemplates.Delete("/:id", waTemplateHandler.Delete)

	// Audit logs
	auditLogs := protected.Group("/audit-logs", middleware.RoleGuard("superadmin"))
	auditLogs.Get("/", auditLogHandler.List)

	// Report routes
	reports := protected.Group("/reports", middleware.PermissionGuard("reports.view"))
	reports.Get("/revenue", reportHandler.GetRevenueReport)
	reports.Get("/customers", reportHandler.GetCustomerGrowth)
	reports.Get("/customer-growth", reportHandler.GetCustomerGrowth)
	reports.Get("/payments", reportHandler.GetPaymentBreakdown)
	reports.Get("/payment-breakdown", reportHandler.GetPaymentBreakdown)
	reports.Get("/collection-rate", reportHandler.GetCollectionRate)
	reports.Get("/profit-loss", reportHandler.GetProfitLoss)
	reports.Get("/vouchers", reportHandler.GetVoucherSales)
	reports.Get("/voucher-sales", reportHandler.GetVoucherSales)
	reports.Get("/export/revenue/excel", middleware.PermissionGuard("reports.export"), exportHandler.ExportRevenueExcel)
	reports.Get("/export/revenue/pdf", middleware.PermissionGuard("reports.export"), exportHandler.ExportRevenuePDF)
	reports.Get("/export/customer-growth/excel", middleware.PermissionGuard("reports.export"), exportHandler.ExportCustomerGrowthExcel)
	reports.Get("/export/customer-growth/pdf", middleware.PermissionGuard("reports.export"), exportHandler.ExportCustomerGrowthPDF)
	reports.Get("/export/profit-loss/excel", middleware.PermissionGuard("reports.export"), exportHandler.ExportProfitLossExcel)
	reports.Get("/export/profit-loss/pdf", middleware.PermissionGuard("reports.export"), exportHandler.ExportProfitLossPDF)

	// Reseller routes
	resellers := protected.Group("/resellers", middleware.PermissionGuard("resellers.view"))
	resellers.Get("/", resellerHandler.List)
	resellers.Post("/", middleware.PermissionGuard("resellers.create"), resellerHandler.Create)
	resellers.Get("/:id", resellerHandler.GetByID)
	resellers.Put("/:id", middleware.PermissionGuard("resellers.edit"), resellerHandler.Update)
	resellers.Delete("/:id", middleware.PermissionGuard("resellers.delete"), resellerHandler.Delete)
	resellers.Get("/:id/commissions", resellerHandler.ListCommissions)
	resellers.Post("/:id/commissions", middleware.PermissionGuard("resellers.edit"), resellerHandler.AddCommission)
	resellers.Post("/:id/commissions/pay-all", middleware.PermissionGuard("resellers.edit"), resellerHandler.PayAllPending)
	resellers.Get("/:id/commission-summary", resellerHandler.GetCommissionSummary)
	resellers.Get("/:id/customers", resellerHandler.ListCustomers)
	resellers.Post("/commissions/:commissionId/pay", middleware.PermissionGuard("resellers.edit"), resellerHandler.PayCommission)

	// Dashboard
	dashboard := protected.Group("/dashboard", middleware.PermissionGuard("dashboard.view"))
	dashboard.Get("/stats", dashboardHandler.GetStats)
	dashboard.Get("/revenue", dashboardHandler.GetMonthlyRevenue)
	dashboard.Get("/revenue/rolling", dashboardHandler.GetRollingRevenue)

	// Reward / Referral routes
	rewards := protected.Group("/rewards", middleware.PermissionGuard("rewards.view"))
	rewards.Get("/", rewardHandler.ListRewards)
	rewards.Post("/", middleware.PermissionGuard("rewards.create"), rewardHandler.CreateReward)
	rewards.Get("/stats", rewardHandler.GetRewardStats)
	rewards.Get("/dashboard", rewardHandler.GetRewardDashboard)
	rewards.Get("/:id", rewardHandler.GetReward)
	rewards.Put("/:id", middleware.PermissionGuard("rewards.edit"), rewardHandler.UpdateReward)
	rewards.Delete("/:id", middleware.PermissionGuard("rewards.delete"), rewardHandler.DeleteReward)

	referrals := protected.Group("/referrals", middleware.PermissionGuard("rewards.view"))
	referrals.Get("/", rewardHandler.ListReferrals)
	referrals.Get("/:id", rewardHandler.GetReferral)
	referrals.Post("/:id/reward", middleware.PermissionGuard("rewards.edit"), rewardHandler.MarkRewarded)

	rewardClaims := protected.Group("/reward-claims", middleware.PermissionGuard("rewards.view"))
	rewardClaims.Get("/", rewardHandler.ListClaims)
	rewardClaims.Post("/:claimId/apply", middleware.PermissionGuard("rewards.edit"), rewardHandler.ApplyClaim)
	rewardClaims.Get("/balance/:id", rewardHandler.GetCustomerBalance)

	// Tenant management routes (superadmin — cross-tenant CRUD)
	tenants := v1.Group("/tenants", authMiddleware.Handle(), middleware.RoleGuard("superadmin"))
	tenants.Get("/", tenantHandler.List)
	tenants.Post("/", tenantHandler.Create)
	tenants.Get("/:id", tenantHandler.GetByID)
	tenants.Put("/:id", tenantHandler.AdminUpdate)
	tenants.Post("/:id/approve", tenantHandler.Approve)
	tenants.Post("/:id/reset-password", adminHandler.ResetTenantPassword)
	tenants.Delete("/:id", tenantHandler.Delete)

	// SuperAdmin (Pengelola) routes — cross-tenant management
	admin := v1.Group("/admin", authMiddleware.Handle(), middleware.RoleGuard("superadmin"))
	admin.Get("/dashboard", adminHandler.GetDashboardStats)
	admin.Get("/tenants", adminHandler.GetTenantStats)
	admin.Get("/routers", adminHandler.GetAllRouters)
	admin.Get("/customers", adminHandler.GetTenantCustomerCounts)
	admin.Get("/revenue/rolling", adminHandler.GetRollingRevenue)
	admin.Get("/revenue/subscription", adminHandler.GetSubscriptionRevenue)

	// SuperAdmin Settings routes
	admin.Get("/settings", saSettingsHandler.GetSettings)
	admin.Put("/settings", saSettingsHandler.UpdateSettings)
	admin.Post("/settings/test-pg", saSettingsHandler.TestPGConnection)
	admin.Get("/webhook-urls", saSettingsHandler.GetWebhookURLs)

	// SuperAdmin WhatsApp session routes (dedicated, uses "superadmin" as session ID)
	admin.Post("/wa/start", saSettingsHandler.StartWASession)
	admin.Get("/wa/status", saSettingsHandler.GetWASessionStatus)
	admin.Get("/wa/qr", saSettingsHandler.GetWAQR)
	admin.Delete("/wa/stop", saSettingsHandler.StopWASession)

	// SuperAdmin Subscription Plan management
	admin.Get("/subscription/plans", adminHandler.ListSubscriptionPlans)
	admin.Post("/subscription/plans", adminHandler.CreateSubscriptionPlan)
	admin.Put("/subscription/plans/:id", adminHandler.UpdateSubscriptionPlan)
	admin.Delete("/subscription/plans/:id", adminHandler.DeleteSubscriptionPlan)

	// SuperAdmin Subscription Reminder templates
	admin.Get("/subscription/reminders", adminHandler.ListSubscriptionReminders)
	admin.Put("/subscription/reminders/:id", adminHandler.UpdateSubscriptionReminder)

	// SuperAdmin Subscription Order CRUD
	admin.Get("/subscription/orders", adminHandler.ListSubOrders)
	admin.Get("/subscription/orders/:id", adminHandler.GetSubOrder)
	admin.Post("/subscription/orders", adminHandler.CreateSubOrder)
	admin.Put("/subscription/orders/:id", adminHandler.UpdateSubOrder)
	admin.Delete("/subscription/orders/:id", adminHandler.DeleteSubOrder)

}
