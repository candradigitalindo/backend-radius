package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"

	"github.com/candrasyahputra/radius-server/internal/config"
	"github.com/candrasyahputra/radius-server/internal/pkg/cache"
	"github.com/candrasyahputra/radius-server/internal/pkg/database"
	"github.com/candrasyahputra/radius-server/internal/pkg/genieacs"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
	"github.com/candrasyahputra/radius-server/internal/service"
	"github.com/candrasyahputra/radius-server/internal/worker"
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

	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Host + ":" + cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// Repositories
	customerRepo := repository.NewCustomerRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	invoiceRepo := repository.NewInvoiceRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	reminderRepo := repository.NewReminderRepository(db)
	routerRepo := repository.NewRouterRepository(db)
	ipamRepo := repository.NewIPAMRepository(db)
	ontRepo := repository.NewONTRepository(db)
	rewardRepo := repository.NewRewardRepository(db)

	// WhatsApp client
	waClient := whatsapp.NewClient(cfg.WhatsApp)

	// Services
	invoiceService := service.NewInvoiceService(invoiceRepo, paymentRepo, customerRepo)
	invoiceService.WithWAClient(waClient).WithReminderRepo(reminderRepo)
	rewardService := service.NewRewardService(rewardRepo).WithInvoice(invoiceRepo)
	invoiceService.WithReward(rewardService)
	customerService := service.NewCustomerService(customerRepo, sessionRepo)
	reminderService := service.NewReminderService(reminderRepo, invoiceRepo, customerRepo, tenantRepo, waClient)
	routerService := service.NewRouterService(routerRepo, sessionRepo, nil)

	// GenieACS service (untuk ONT auto-provisioning retry)
	genieacsClient := genieacs.NewClient(cfg.GenieACS)
	genieacsService := service.NewGenieACSService(genieacsClient, ontRepo, customerRepo)

	// Task handlers
	handlers := &worker.Handlers{
		DB:              db,
		InvoiceService:  invoiceService,
		CustomerService: customerService,
		ReminderService: reminderService,
		TenantRepo:      tenantRepo,
		InvoiceRepo:     invoiceRepo,
		ReminderRepo:    reminderRepo,
		RouterRepo:      routerRepo,
		SessionRepo:     sessionRepo,
		IPAMRepo:        ipamRepo,
		RouterService:   routerService,
		RewardService:   rewardService,
		WAClient:        waClient,
		GenieACSService: genieacsService,
		ONTRepo:         ontRepo,
	}

	// Asynq server
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"default": 6,
			"low":     4,
		},
		ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
			log.Printf("[worker] ERROR task=%s err=%v", task.Type(), err)
		}),
	})

	// Asynq client
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// Mux
	mux := asynq.NewServeMux()
	handlers.Register(mux)
	worker.RegisterSchedulerHandlers(mux, tenantRepo, client)

	// Scheduler
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatalf("Failed to load timezone: %v", err)
	}
	scheduler := asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Location: loc,
	})

	schedulerCfg := worker.SchedulerConfig{
		InvoiceCron:       getEnv("CRON_INVOICE", "0 8 * * *"),
		IsolirCron:        getEnv("CRON_ISOLIR", "0 6 * * *"),
		ReminderCron:      getEnv("CRON_REMINDER", "0 10 * * *"),
		ExpirePayCron:     getEnv("CRON_EXPIRE_PAY", "*/30 * * * *"),
		RouterMonitorCron: getEnv("CRON_ROUTER_MONITOR", "*/5 * * * *"),
		CleanSessionsCron: getEnv("CRON_CLEAN_SESSIONS", "*/5 * * * *"),
		ONTProvisionCron:  getEnv("CRON_ONT_PROVISION", "*/2 * * * *"),
		ONTDiscoverCron:   getEnv("CRON_ONT_DISCOVER", "*/5 * * * *"),
		ONTAutoMatchCron:  getEnv("CRON_ONT_AUTO_MATCH", "*/5 * * * *"),
		ExpireRewardsCron: getEnv("CRON_EXPIRE_REWARDS", "0 2 * * *"),
	}

	if err := worker.SetupScheduler(scheduler, tenantRepo, schedulerCfg); err != nil {
		log.Fatalf("Failed to setup scheduler: %v", err)
	}

	go func() {
		if err := scheduler.Run(); err != nil {
			log.Fatalf("Scheduler error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		fmt.Println("\nShutting down worker...")
		srv.Shutdown()
		scheduler.Shutdown()
	}()

	fmt.Println("Worker starting...")
	if err := srv.Run(mux); err != nil {
		log.Fatalf("Worker server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
