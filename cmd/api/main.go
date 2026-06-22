package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/middleware"
	"github.com/candrasyahputra/radius-server/internal/pkg/cache"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	radiuspkg "github.com/candrasyahputra/radius-server/internal/pkg/radius"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/router"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db := database.Connect(cfg)
	defer db.Close()

	rdb := cache.Connect(cfg)
	defer rdb.Close()

	// Repositories
	customerRepo := repository.NewCustomerRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	routerRepo := repository.NewRouterRepository(db)
	voucherRepo := repository.NewVoucherRepository(db)
	ipamRepo := repository.NewIPAMRepository(db)

	// RADIUS Server
	radiusSrv := radiuspkg.NewServer(customerRepo, sessionRepo, routerRepo, voucherRepo, ipamRepo, cfg.RADIUS.Secret)

	go func() {
		if err := radiusSrv.StartAuth(cfg.RADIUS.ListenAuth); err != nil {
			log.Fatalf("RADIUS Auth server error: %v", err)
		}
	}()

	go func() {
		if err := radiusSrv.StartAcct(cfg.RADIUS.ListenAcct); err != nil {
			log.Fatalf("RADIUS Acct server error: %v", err)
		}
	}()

	// Fiber HTTP server
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
		// Behind the nginx reverse proxy every request arrives from the proxy's
		// container IP, so c.IP() must read the real client IP from X-Forwarded-For.
		// Without this, per-IP rate limiters key every user under the proxy IP and
		// trip a shared/global 429 on login. Only trust the header from the proxy.
		ProxyHeader:             fiber.HeaderXForwardedFor,
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
	})

	middleware.SetupGlobal(app, cfg.CORS.AllowedOrigins)

	router.Setup(app, &router.Dependencies{
		Config: cfg,
		DB:     db,
		Redis:  rdb,
	})

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down server...")
		_ = app.Shutdown()
	}()

	addr := ":" + cfg.App.Port
	fmt.Printf("Server starting on %s (env: %s)\n", addr, cfg.App.Env)

	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
	})
}
