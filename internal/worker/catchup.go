package worker

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// cronLastRunKey is where MarkDailyRun/RunCatchUp track the last WIB calendar
// date a daily scheduler task actually ran, keyed by asynq task type.
func cronLastRunKey(taskType string) string {
	return "cron:lastrun:" + taskType
}

// jakartaLocation returns Asia/Jakarta, falling back to UTC if the tzdata
// lookup fails (matches the fallback already used elsewhere in this package).
func jakartaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.UTC
	}
	return loc
}

// MarkDailyRun records that a daily scheduler task ran today (WIB). Call this
// at the end of a scheduler:* handler after its fan-out completes, so a
// restart later the same day doesn't trigger a duplicate catch-up run.
func MarkDailyRun(ctx context.Context, rdb *redis.Client, taskType string) {
	if rdb == nil {
		return
	}
	today := time.Now().In(jakartaLocation()).Format("2006-01-02")
	if err := rdb.Set(ctx, cronLastRunKey(taskType), today, 48*time.Hour).Err(); err != nil {
		log.Printf("[scheduler] gagal menandai lastrun untuk %s: %v", taskType, err)
	}
}

// catchUpJob is one daily cron job eligible for missed-run catch-up.
type catchUpJob struct {
	name     string // untuk log
	taskType string // asynq task type yang di-enqueue ulang
	cronExpr string // ekspresi cron "menit jam * * *" dari SchedulerConfig
}

// parseDailyCron extracts hour/minute from a standard 5-field daily cron
// expression ("M H * * *"). Returns ok=false for anything else (interval
// crons like */5, or weekday-restricted crons) since catch-up only makes
// sense for once-a-day jobs.
func parseDailyCron(expr string) (hour, minute int, ok bool) {
	fields := strings.Fields(expr)
	if len(fields) != 5 || fields[2] != "*" || fields[3] != "*" || fields[4] != "*" {
		return 0, 0, false
	}
	m, errM := strconv.Atoi(fields[0])
	h, errH := strconv.Atoi(fields[1])
	if errM != nil || errH != nil {
		return 0, 0, false
	}
	return h, m, true
}

// RunCatchUp checks the daily billing/subscription cron jobs against their
// last-run marker and enqueues any that were due earlier today but never
// fired — the case when the worker container was down or restarting at the
// scheduled hour (asynq's scheduler has no built-in catch-up: a missed tick
// is simply skipped, not deferred). Call once at worker startup, after the
// scheduler and mux are set up but before serving.
func RunCatchUp(ctx context.Context, rdb *redis.Client, client *asynq.Client, cfg SchedulerConfig) {
	if rdb == nil || client == nil {
		return
	}

	gracePeriodCron := cfg.GracePeriodCron
	if gracePeriodCron == "" {
		gracePeriodCron = "0 9 * * *"
	}
	subExpiryCron := cfg.SubExpiryCron
	if subExpiryCron == "" {
		subExpiryCron = "0 8 * * *"
	}

	jobs := []catchUpJob{
		{"auto-isolir", "scheduler:auto_isolir_all", cfg.IsolirCron},
		{"generate-invoices", "scheduler:generate_invoices_all", cfg.InvoiceCron},
		{"grace-period-check", "scheduler:grace_period_check_all", gracePeriodCron},
		{"trigger-reminders", "scheduler:trigger_reminders_all", cfg.ReminderCron},
		{"sub-expiry-check", TaskSubExpiryCheck, subExpiryCron},
	}

	loc := jakartaLocation()
	now := time.Now().In(loc)
	today := now.Format("2006-01-02")

	for _, j := range jobs {
		hour, minute, ok := parseDailyCron(j.cronExpr)
		if !ok {
			continue // bukan cron harian sederhana — lewati, biarkan asynq scheduler yang menangani
		}

		scheduledToday := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if now.Before(scheduledToday) {
			continue // jadwal hari ini belum lewat, tidak perlu catch-up
		}

		lastRun, err := rdb.Get(ctx, cronLastRunKey(j.taskType)).Result()
		if err != nil && err != redis.Nil {
			log.Printf("[catchup] gagal cek status %s: %v", j.name, err)
			continue
		}
		if lastRun == today {
			continue // sudah jalan hari ini (baik lewat cron normal maupun catch-up sebelumnya)
		}

		if _, err := client.Enqueue(asynq.NewTask(j.taskType, nil), asynq.Queue("default")); err != nil {
			log.Printf("[catchup] gagal enqueue %s: %v", j.name, err)
			continue
		}
		log.Printf("[catchup] %s (jadwal %02d:%02d WIB) terlewat hari ini — worker baru start, dijalankan sekarang", j.name, hour, minute)
	}
}
