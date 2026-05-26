package service

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/whatsapp"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

var (
	ErrBroadcastNotFound = errors.New("Broadcast tidak ditemukan")
)

type BroadcastService struct {
	broadcastRepo repository.BroadcastRepository
	customerRepo  repository.CustomerRepository
	waClient      *whatsapp.Client
	resumeMu      sync.Mutex
}

func NewBroadcastService(
	broadcastRepo repository.BroadcastRepository,
	customerRepo repository.CustomerRepository,
	waClient *whatsapp.Client,
) *BroadcastService {
	return &BroadcastService{
		broadcastRepo: broadcastRepo,
		customerRepo:  customerRepo,
		waClient:      waClient,
	}
}

type SendBroadcastInput struct {
	TenantID    string
	Type        string
	Title       string
	Message     string
	ImageURL    string
	Target      string
	SentBy      string
	CleanupFunc func()
}

func (s *BroadcastService) Send(ctx context.Context, input SendBroadcastInput) (*model.Broadcast, error) {
	if input.Target == "" {
		input.Target = "all"
	}

	// Determine which statuses to fetch
	var statuses []string
	switch input.Target {
	case "active":
		statuses = []string{"active"}
	case "inactive":
		statuses = []string{"inactive"}
	case "isolated":
		statuses = []string{"isolated"}
	default: // "all"
		statuses = []string{"active", "inactive", "isolated"}
	}

	var allCustomers []model.Customer
	for _, status := range statuses {
		page := 1
		const perPage = 500
		for {
			filter := repository.CustomerFilter{Status: status, Page: page, PerPage: perPage}
			customers, _, err := s.customerRepo.List(ctx, input.TenantID, filter)
			if err != nil {
				return nil, err
			}
			allCustomers = append(allCustomers, customers...)
			if len(customers) < perPage {
				break
			}
			page++
		}
	}

	// Count actual phones (exclude customers without phone numbers)
	phoneCount := 0
	for _, c := range allCustomers {
		if c.Phone != "" {
			phoneCount++
		}
	}

	broadcast := &model.Broadcast{
		TenantID:     input.TenantID,
		Type:         input.Type,
		Title:        input.Title,
		Message:      input.Message,
		ImageURL:     input.ImageURL,
		Target:       input.Target,
		Status:       "sending",
		TotalSent:    phoneCount,
		TotalSuccess: 0,
		TotalFailed:  0,
		TotalPending: 0,
		SentBy:       input.SentBy,
	}

	if err := s.broadcastRepo.Create(ctx, broadcast); err != nil {
		return nil, err
	}

	go s.processBroadcast(broadcast, allCustomers, input.CleanupFunc)

	return s.broadcastRepo.FindByID(ctx, input.TenantID, broadcast.ID)
}

func (s *BroadcastService) processBroadcast(broadcast *model.Broadcast, customers []model.Customer, cleanup func()) {
	phones := make([]string, 0, len(customers))
	for _, c := range customers {
		if c.Phone != "" {
			phones = append(phones, c.Phone)
		}
	}

	if len(phones) == 0 {
		broadcast.Status = "completed"
		_ = s.broadcastRepo.UpdateStats(context.Background(), broadcast)
		if cleanup != nil {
			cleanup()
		}
		return
	}

	s.sendAndUpdateBroadcast(broadcast, phones)
	if cleanup != nil {
		cleanup()
	}
}

func (s *BroadcastService) sendAndUpdateBroadcast(broadcast *model.Broadcast, phones []string) {
	result, err := s.waClient.SendBroadcastWithImage(
		context.Background(), broadcast.TenantID, phones, broadcast.Message, broadcast.ImageURL,
	)
	if err != nil {
		log.Printf("[broadcast] WA send failed for broadcast %s: %v", broadcast.ID, err)
		broadcast.TotalFailed = len(phones)
		broadcast.Status = "failed"
		_ = s.broadcastRepo.UpdateStats(context.Background(), broadcast)
		return
	}

	broadcast.TotalSuccess += result.Success
	broadcast.TotalFailed += result.Failed

	if result.Pending > 0 && len(result.PendingPhones) > 0 {
		broadcast.TotalPending = result.Pending
		broadcast.PendingPhones = strings.Join(result.PendingPhones, ",")
		broadcast.Status = "pending"
		log.Printf("[broadcast] %s: sent %d, pending %d (will resume tomorrow)",
			broadcast.ID, result.Success, result.Pending)
	} else {
		broadcast.TotalPending = 0
		broadcast.PendingPhones = ""
		broadcast.Status = "completed"
	}

	if updateErr := s.broadcastRepo.UpdateStats(context.Background(), broadcast); updateErr != nil {
		log.Printf("[broadcast] failed to update stats for %s: %v", broadcast.ID, updateErr)
	}
}

// ResumePendingBroadcasts finds all broadcasts with status 'pending' and resumes sending.
// Runs synchronously to prevent duplicate sends from concurrent scheduler ticks.
func (s *BroadcastService) ResumePendingBroadcasts() {
	s.resumeMu.Lock()
	defer s.resumeMu.Unlock()

	pending, err := s.broadcastRepo.ListPending(context.Background())
	if err != nil {
		log.Printf("[broadcast-scheduler] failed to list pending broadcasts: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Printf("[broadcast-scheduler] resuming %d pending broadcasts", len(pending))

	for i := range pending {
		b := &pending[i]
		if b.PendingPhones == "" {
			b.Status = "completed"
			b.TotalPending = 0
			_ = s.broadcastRepo.UpdateStats(context.Background(), b)
			continue
		}

		phones := strings.Split(b.PendingPhones, ",")
		log.Printf("[broadcast-scheduler] resuming broadcast %s: %d phones pending", b.ID, len(phones))
		s.sendAndUpdateBroadcast(b, phones)
	}
}

// StartScheduler starts a background goroutine that checks for pending broadcasts periodically.
func (s *BroadcastService) StartScheduler() {
	go func() {
		// Initial check on startup
		time.Sleep(10 * time.Second)
		s.ResumePendingBroadcasts()

		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			s.ResumePendingBroadcasts()
		}
	}()
	log.Println("[broadcast-scheduler] started - checks pending broadcasts every hour")
}

func (s *BroadcastService) GetByID(ctx context.Context, tenantID, broadcastID string) (*model.Broadcast, error) {
	b, err := s.broadcastRepo.FindByID(ctx, tenantID, broadcastID)
	if err != nil {
		return nil, err
	}
	if b == nil {
		return nil, ErrBroadcastNotFound
	}
	return b, nil
}

func (s *BroadcastService) Delete(ctx context.Context, tenantID, broadcastID string) error {
	b, err := s.broadcastRepo.FindByID(ctx, tenantID, broadcastID)
	if err != nil {
		return err
	}
	if b == nil {
		return ErrBroadcastNotFound
	}
	return s.broadcastRepo.Delete(ctx, tenantID, broadcastID)
}

func (s *BroadcastService) List(ctx context.Context, tenantID string, filter repository.BroadcastFilter) ([]model.Broadcast, int, error) {
	return s.broadcastRepo.List(ctx, tenantID, filter)
}
