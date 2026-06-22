package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

// ErrNoUnpaidInvoice is returned when a reward credit cannot be applied because the
// customer has no outstanding invoice to discount.
var ErrNoUnpaidInvoice = errors.New("Tidak ada tagihan belum dibayar untuk menerapkan reward")

type RewardService struct {
	rewardRepo  repository.RewardRepository
	invoiceRepo repository.InvoiceRepository
}

func NewRewardService(rewardRepo repository.RewardRepository) *RewardService {
	return &RewardService{rewardRepo: rewardRepo}
}

func (s *RewardService) WithInvoice(invoiceRepo repository.InvoiceRepository) *RewardService {
	s.invoiceRepo = invoiceRepo
	return s
}

// Reward CRUD

func (s *RewardService) CreateReward(ctx context.Context, reward *model.Reward) error {
	return s.rewardRepo.CreateReward(ctx, reward)
}

func (s *RewardService) GetReward(ctx context.Context, tenantID, rewardID string) (*model.Reward, error) {
	return s.rewardRepo.GetReward(ctx, tenantID, rewardID)
}

func (s *RewardService) UpdateReward(ctx context.Context, reward *model.Reward) error {
	return s.rewardRepo.UpdateReward(ctx, reward)
}

func (s *RewardService) DeleteReward(ctx context.Context, tenantID, rewardID string) error {
	return s.rewardRepo.DeleteReward(ctx, tenantID, rewardID)
}

func (s *RewardService) ListRewards(ctx context.Context, tenantID string, activeOnly bool, page, perPage int) ([]model.Reward, int, error) {
	return s.rewardRepo.ListRewards(ctx, tenantID, activeOnly, page, perPage)
}

// Referral

func (s *RewardService) CreateReferral(ctx context.Context, ref *model.Referral) error {
	return s.rewardRepo.CreateReferral(ctx, ref)
}

func (s *RewardService) GetReferral(ctx context.Context, tenantID, referralID string) (*model.Referral, error) {
	return s.rewardRepo.GetReferral(ctx, tenantID, referralID)
}

func (s *RewardService) GetReferralByCode(ctx context.Context, tenantID, code string) (*model.Referral, error) {
	return s.rewardRepo.GetReferralByCode(ctx, tenantID, code)
}

func (s *RewardService) ListReferrals(ctx context.Context, tenantID string, filter repository.ReferralFilter) ([]model.Referral, int, error) {
	return s.rewardRepo.ListReferrals(ctx, tenantID, filter)
}

func (s *RewardService) UpdateReferralStatus(ctx context.Context, tenantID, referralID, status string) error {
	return s.rewardRepo.UpdateReferralStatus(ctx, tenantID, referralID, status)
}

func (s *RewardService) MarkRewarded(ctx context.Context, tenantID, referralID string) error {
	return s.rewardRepo.MarkRewarded(ctx, tenantID, referralID)
}

func (s *RewardService) GetCustomerReferralCode(ctx context.Context, tenantID, customerID string) (string, error) {
	return s.rewardRepo.GetCustomerReferralCode(ctx, tenantID, customerID)
}

// Claims

func (s *RewardService) CreateClaim(ctx context.Context, claim *model.RewardClaim) error {
	return s.rewardRepo.CreateClaim(ctx, claim)
}

func (s *RewardService) ListClaims(ctx context.Context, tenantID string, filter repository.ClaimFilter) ([]model.RewardClaim, int, error) {
	return s.rewardRepo.ListClaims(ctx, tenantID, filter)
}

// ApplyClaim redeems a pending reward claim by crediting its amount as a discount
// on the customer's oldest unpaid invoice, then marks the claim applied.
func (s *RewardService) ApplyClaim(ctx context.Context, tenantID, claimID string) error {
	claim, err := s.rewardRepo.GetClaim(ctx, tenantID, claimID)
	if err != nil {
		return err
	}
	if claim == nil {
		return errors.New("Klaim tidak ditemukan")
	}
	if claim.Status != "pending" {
		return errors.New("Klaim sudah diproses")
	}

	// Apply the reward as a credit/discount on the customer's oldest unpaid invoice.
	if s.invoiceRepo != nil && claim.Amount > 0 {
		inv, err := s.invoiceRepo.FindOldestUnpaidByCustomer(ctx, tenantID, claim.CustomerID)
		if err != nil {
			return err
		}
		if inv == nil {
			return ErrNoUnpaidInvoice
		}
		credit := claim.Amount
		if credit > inv.TotalAmount {
			credit = inv.TotalAmount
		}
		if credit > 0 {
			if err := s.invoiceRepo.ApplyCredit(ctx, tenantID, inv.ID, credit); err != nil {
				return err
			}
			log.Printf("[reward] applied credit %d to invoice %s (customer %s, claim %s)", credit, inv.InvoiceNumber, claim.CustomerID, claimID)
		}
	}

	return s.rewardRepo.ApplyClaim(ctx, tenantID, claimID)
}

func (s *RewardService) GetCustomerBalance(ctx context.Context, tenantID, customerID string) (int64, error) {
	return s.rewardRepo.GetCustomerBalance(ctx, tenantID, customerID)
}

func (s *RewardService) GetRewardStats(ctx context.Context, tenantID string) (*repository.RewardStats, error) {
	return s.rewardRepo.GetRewardStats(ctx, tenantID)
}

func (s *RewardService) FindActiveRewardByType(ctx context.Context, tenantID, rewardType string) (*model.Reward, error) {
	return s.rewardRepo.FindActiveRewardByType(ctx, tenantID, rewardType)
}

func (s *RewardService) GetRewardDashboard(ctx context.Context, tenantID string, months int) (*repository.RewardDashboard, error) {
	return s.rewardRepo.GetRewardDashboard(ctx, tenantID, months)
}

// ProcessRewardOnPayment is called after an invoice is marked paid.
// It handles:
//  1. Auto-qualify pending referral for the customer (referred) → create claim for referrer.
//  2. Auto-check loyalty rewards based on total paid invoices.
func (s *RewardService) ProcessRewardOnPayment(ctx context.Context, tenantID, customerID string) {
	// 1. Referral qualification
	ref, err := s.rewardRepo.FindPendingReferralByReferred(ctx, tenantID, customerID)
	if err != nil {
		log.Printf("[reward] find pending referral for customer %s: %v", customerID, err)
	} else if ref != nil {
		if err := s.rewardRepo.QualifyReferral(ctx, tenantID, ref.ID); err != nil {
			log.Printf("[reward] qualify referral %s: %v", ref.ID, err)
		} else {
			// Fetch reward config to determine claim amount
			reward, err := s.rewardRepo.GetReward(ctx, tenantID, ref.RewardID)
			if err != nil || reward == nil {
				log.Printf("[reward] get reward %s for referral claim: %v", ref.RewardID, err)
			} else {
				claim := &model.RewardClaim{
					TenantID:   tenantID,
					CustomerID: ref.ReferrerID,
					RewardID:   reward.ID,
					ReferralID: &ref.ID,
					Amount:     reward.Value,
					Type:       "balance_credit",
					Status:     "pending",
				}
				if reward.EndDate != nil {
					exp := reward.EndDate.Add(30 * 24 * time.Hour)
					claim.ExpiresAt = &exp
				}
				if err := s.rewardRepo.CreateClaim(ctx, claim); err != nil {
					log.Printf("[reward] create referral claim for referrer %s: %v", ref.ReferrerID, err)
				}
			}
			// Mark referral as rewarded
			if err := s.rewardRepo.MarkRewarded(ctx, tenantID, ref.ID); err != nil {
				log.Printf("[reward] mark referral %s rewarded: %v", ref.ID, err)
			}
		}
	}

	// 2. Loyalty reward check
	if s.invoiceRepo == nil {
		return
	}
	loyaltyRewards, err := s.rewardRepo.FindActiveLoyaltyRewards(ctx, tenantID)
	if err != nil || len(loyaltyRewards) == 0 {
		return
	}
	paidCount, err := s.invoiceRepo.CountPaidInvoices(ctx, tenantID, customerID)
	if err != nil {
		log.Printf("[reward] count paid invoices for customer %s: %v", customerID, err)
		return
	}
	for _, rw := range loyaltyRewards {
		if rw.MinInvoices <= 0 || paidCount < rw.MinInvoices {
			continue
		}
		already, err := s.rewardRepo.HasLoyaltyClaim(ctx, tenantID, customerID, rw.ID)
		if err != nil || already {
			continue
		}
		claim := &model.RewardClaim{
			TenantID:   tenantID,
			CustomerID: customerID,
			RewardID:   rw.ID,
			Amount:     rw.Value,
			Type:       "balance_credit",
			Status:     "pending",
		}
		if err := s.rewardRepo.CreateClaim(ctx, claim); err != nil {
			log.Printf("[reward] create loyalty claim for customer %s reward %s: %v", customerID, rw.ID, err)
		}
	}
}

// ExpireOldClaims expires pending claims that have passed their expires_at date.
func (s *RewardService) ExpireOldClaims(ctx context.Context) (int, error) {
	return s.rewardRepo.ExpireOldClaims(ctx)
}
