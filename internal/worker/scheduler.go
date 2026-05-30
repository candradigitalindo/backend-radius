package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"

	"github.com/candrasyahputra/radius-server/internal/repository"
)

type SchedulerConfig struct {
	InvoiceCron        string
	IsolirCron         string
	ReminderCron       string
	ExpirePayCron      string
	RouterMonitorCron  string
	CleanSessionsCron  string
	ONTProvisionCron   string
	ONTDiscoverCron    string
	ONTAutoMatchCron   string
	ExpireRewardsCron  string
	SubExpiryCron      string
}

func invoiceTargetPeriod(now time.Time) (month, year int) {
	target := now.AddDate(0, 0, 7)
	return int(target.Month()), target.Year()
}

func SetupScheduler(scheduler *asynq.Scheduler, tenantRepo repository.TenantRepository, cfg SchedulerConfig) error {
	if _, err := scheduler.Register(cfg.InvoiceCron, asynq.NewTask("scheduler:generate_invoices_all", nil)); err != nil {
		return fmt.Errorf("register invoice scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.IsolirCron, asynq.NewTask("scheduler:auto_isolir_all", nil)); err != nil {
		return fmt.Errorf("register isolir scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.ReminderCron, asynq.NewTask("scheduler:trigger_reminders_all", nil)); err != nil {
		return fmt.Errorf("register reminder scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.ExpirePayCron, asynq.NewTask(TaskExpirePayments, nil)); err != nil {
		return fmt.Errorf("register expire-payments scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.RouterMonitorCron, asynq.NewTask("scheduler:router_monitor_all", nil)); err != nil {
		return fmt.Errorf("register router-monitor scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.CleanSessionsCron, asynq.NewTask("scheduler:clean_sessions_all", nil)); err != nil {
		return fmt.Errorf("register clean-sessions scheduler: %w", err)
	}

	if _, err := scheduler.Register(cfg.ONTProvisionCron, asynq.NewTask(TaskONTProvisionRetry, nil)); err != nil {
		return fmt.Errorf("register ont-provision-retry scheduler: %w", err)
	}

	if cfg.ONTDiscoverCron != "" {
		if _, err := scheduler.Register(cfg.ONTDiscoverCron, asynq.NewTask(TaskONTDiscover, nil)); err != nil {
			return fmt.Errorf("register ont-discover scheduler: %w", err)
		}
	}

	if cfg.ONTAutoMatchCron != "" {
		if _, err := scheduler.Register(cfg.ONTAutoMatchCron, asynq.NewTask(TaskONTAutoMatch, nil)); err != nil {
			return fmt.Errorf("register ont-auto-match scheduler: %w", err)
		}
	}

	expireRewardsCron := cfg.ExpireRewardsCron
	if expireRewardsCron == "" {
		expireRewardsCron = "0 2 * * *" // default: daily at 02:00
	}
	if _, err := scheduler.Register(expireRewardsCron, asynq.NewTask(TaskExpireRewardClaims, nil)); err != nil {
		return fmt.Errorf("register expire-reward-claims scheduler: %w", err)
	}

	subExpiryCron := cfg.SubExpiryCron
	if subExpiryCron == "" {
		subExpiryCron = "0 8 * * *" // daily at 08:00 WIB
	}
	if _, err := scheduler.Register(subExpiryCron, asynq.NewTask(TaskSubExpiryCheck, nil)); err != nil {
		return fmt.Errorf("register sub-expiry-check scheduler: %w", err)
	}

	return nil
}

func RegisterSchedulerHandlers(mux *asynq.ServeMux, tenantRepo repository.TenantRepository, client *asynq.Client) {
	mux.HandleFunc("scheduler:generate_invoices_all", func(ctx context.Context, t *asynq.Task) error {
		return enqueueForAllTenants(ctx, tenantRepo, client, func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error) {
			month, year := invoiceTargetPeriod(time.Now())
			return NewGenerateInvoicesTask(GenerateInvoicesPayload{
				TenantID: tenantID,
				Month:    month,
				Year:     year,
				DueDay:   dueDay,
			})
		})
	})

	mux.HandleFunc("scheduler:auto_isolir_all", func(ctx context.Context, t *asynq.Task) error {
		return enqueueForAllTenants(ctx, tenantRepo, client, func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error) {
			return NewAutoIsolirTask(AutoIsolirPayload{TenantID: tenantID})
		})
	})

	mux.HandleFunc("scheduler:trigger_reminders_all", func(ctx context.Context, t *asynq.Task) error {
		return enqueueForAllTenants(ctx, tenantRepo, client, func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error) {
			return NewTriggerRemindersTask(TriggerRemindersPayload{TenantID: tenantID})
		})
	})

	mux.HandleFunc("scheduler:router_monitor_all", func(ctx context.Context, t *asynq.Task) error {
		return enqueueForAllTenants(ctx, tenantRepo, client, func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error) {
			return NewRouterMonitorTask(RouterMonitorPayload{TenantID: tenantID})
		})
	})

	mux.HandleFunc("scheduler:clean_sessions_all", func(ctx context.Context, t *asynq.Task) error {
		return enqueueForAllTenants(ctx, tenantRepo, client, func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error) {
			return NewCleanSessionsTask(CleanSessionsPayload{TenantID: tenantID})
		})
	})
}

type taskFactory func(tenantID string, billingCycle, dueDay int) (*asynq.Task, error)

func enqueueForAllTenants(ctx context.Context, tenantRepo repository.TenantRepository, client *asynq.Client, factory taskFactory) error {
	active := true
	tenants, _, err := tenantRepo.List(ctx, repository.TenantFilter{
		Active:  &active,
		Page:    1,
		PerPage: 10000,
	})
	if err != nil {
		return fmt.Errorf("list active tenants: %w", err)
	}

	for _, tenant := range tenants {
		task, err := factory(tenant.ID, tenant.BillingCycle, tenant.DueDay)
		if err != nil {
			log.Printf("[scheduler] failed to create task for tenant %s: %v", tenant.ID, err)
			continue
		}

		if _, err := client.EnqueueContext(ctx, task, asynq.Queue("default"), asynq.MaxRetry(3)); err != nil {
			log.Printf("[scheduler] failed to enqueue task for tenant %s: %v", tenant.ID, err)
			continue
		}
	}

	return nil
}
