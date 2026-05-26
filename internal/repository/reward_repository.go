package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/pkg/id"
)

type RewardRepository interface {
	// Reward CRUD
	CreateReward(ctx context.Context, reward *model.Reward) error
	GetReward(ctx context.Context, tenantID, rewardID string) (*model.Reward, error)
	UpdateReward(ctx context.Context, reward *model.Reward) error
	DeleteReward(ctx context.Context, tenantID, rewardID string) error
	ListRewards(ctx context.Context, tenantID string, activeOnly bool, page, perPage int) ([]model.Reward, int, error)

	// Referral
	CreateReferral(ctx context.Context, ref *model.Referral) error
	GetReferral(ctx context.Context, tenantID, referralID string) (*model.Referral, error)
	GetReferralByCode(ctx context.Context, tenantID, code string) (*model.Referral, error)
	ListReferrals(ctx context.Context, tenantID string, filter ReferralFilter) ([]model.Referral, int, error)
	UpdateReferralStatus(ctx context.Context, tenantID, referralID, status string) error
	MarkRewarded(ctx context.Context, tenantID, referralID string) error
	GetCustomerReferralCode(ctx context.Context, tenantID, customerID string) (string, error)

	// Reward Claims
	CreateClaim(ctx context.Context, claim *model.RewardClaim) error
	ListClaims(ctx context.Context, tenantID string, filter ClaimFilter) ([]model.RewardClaim, int, error)
	ApplyClaim(ctx context.Context, tenantID, claimID string) error
	GetCustomerBalance(ctx context.Context, tenantID, customerID string) (int64, error)
	GetRewardStats(ctx context.Context, tenantID string) (*RewardStats, error)

	// Automation helpers
	FindPendingReferralByReferred(ctx context.Context, tenantID, referredID string) (*model.Referral, error)
	QualifyReferral(ctx context.Context, tenantID, referralID string) error
	FindActiveRewardByType(ctx context.Context, tenantID, rewardType string) (*model.Reward, error)
	FindActiveLoyaltyRewards(ctx context.Context, tenantID string) ([]model.Reward, error)
	HasLoyaltyClaim(ctx context.Context, tenantID, customerID, rewardID string) (bool, error)
	ExpireOldClaims(ctx context.Context) (int, error)

	// Dashboard
	GetRewardDashboard(ctx context.Context, tenantID string, months int) (*RewardDashboard, error)
}

type ReferralFilter struct {
	CustomerID string
	Status     string
	Page       int
	PerPage    int
}

type ClaimFilter struct {
	CustomerID string
	Status     string
	Page       int
	PerPage    int
}

type RewardStats struct {
	TotalReferrals     int   `json:"total_referrals"`
	QualifiedReferrals int   `json:"qualified_referrals"`
	TotalRewarded      int64 `json:"total_rewarded"`
	PendingClaims      int   `json:"pending_claims"`
	ActiveRewards      int   `json:"active_rewards"`
}

type RewardMonthlyTrend struct {
	Month         int   `json:"month"`
	Year          int   `json:"year"`
	Referrals     int   `json:"referrals"`
	Qualified     int   `json:"qualified"`
	Claims        int   `json:"claims"`
	ClaimedAmount int64 `json:"claimed_amount"`
}

type TopReferrer struct {
	CustomerID     string `json:"customer_id"`
	CustomerName   string `json:"customer_name"`
	ReferralCount  int    `json:"referral_count"`
	QualifiedCount int    `json:"qualified_count"`
	TotalRewarded  int64  `json:"total_rewarded"`
}

type ClaimTypeStat struct {
	Type        string `json:"type"`
	Status      string `json:"status"`
	Count       int    `json:"count"`
	TotalAmount int64  `json:"total_amount"`
}

type RewardDashboard struct {
	Stats           RewardStats          `json:"stats"`
	MonthlyTrends   []RewardMonthlyTrend `json:"monthly_trends"`
	TopReferrers    []TopReferrer        `json:"top_referrers"`
	RecentReferrals []model.Referral     `json:"recent_referrals"`
	RecentClaims    []model.RewardClaim  `json:"recent_claims"`
	ClaimBreakdown  []ClaimTypeStat      `json:"claim_breakdown"`
}

type rewardRepository struct {
	db *pgxpool.Pool
}

func NewRewardRepository(db *pgxpool.Pool) RewardRepository {
	return &rewardRepository{db: db}
}

// Reward CRUD

func (r *rewardRepository) CreateReward(ctx context.Context, reward *model.Reward) error {
	reward.ID = id.New()
	now := time.Now()
	reward.CreatedAt = now
	reward.UpdatedAt = now

	query := `
		INSERT INTO rewards (id, tenant_id, name, description, type, value, value_type, min_invoices, is_active, start_date, end_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err := r.db.Exec(ctx, query,
		reward.ID, reward.TenantID, reward.Name, reward.Description,
		reward.Type, reward.Value, reward.ValueType, reward.MinInvoices,
		reward.IsActive, reward.StartDate, reward.EndDate,
		reward.CreatedAt, reward.UpdatedAt,
	)
	return err
}

func (r *rewardRepository) GetReward(ctx context.Context, tenantID, rewardID string) (*model.Reward, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(description,''), type, value, value_type, min_invoices, is_active, start_date, end_date, created_at, updated_at
		FROM rewards WHERE id = $1 AND tenant_id = $2
	`
	var rw model.Reward
	err := r.db.QueryRow(ctx, query, rewardID, tenantID).Scan(
		&rw.ID, &rw.TenantID, &rw.Name, &rw.Description,
		&rw.Type, &rw.Value, &rw.ValueType, &rw.MinInvoices,
		&rw.IsActive, &rw.StartDate, &rw.EndDate,
		&rw.CreatedAt, &rw.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rw, nil
}

func (r *rewardRepository) UpdateReward(ctx context.Context, reward *model.Reward) error {
	reward.UpdatedAt = time.Now()
	query := `
		UPDATE rewards SET name = $1, description = $2, type = $3, value = $4, value_type = $5,
		       min_invoices = $6, is_active = $7, start_date = $8, end_date = $9, updated_at = $10
		WHERE id = $11 AND tenant_id = $12
	`
	_, err := r.db.Exec(ctx, query,
		reward.Name, reward.Description, reward.Type, reward.Value, reward.ValueType,
		reward.MinInvoices, reward.IsActive, reward.StartDate, reward.EndDate, reward.UpdatedAt,
		reward.ID, reward.TenantID,
	)
	return err
}

func (r *rewardRepository) DeleteReward(ctx context.Context, tenantID, rewardID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM rewards WHERE id = $1 AND tenant_id = $2`, rewardID, tenantID)
	return err
}

func (r *rewardRepository) ListRewards(ctx context.Context, tenantID string, activeOnly bool, page, perPage int) ([]model.Reward, int, error) {
	if perPage <= 0 {
		perPage = 20
	}
	if page <= 0 {
		page = 1
	}

	where := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	if activeOnly {
		where += " AND is_active = true"
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM rewards "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, COALESCE(description,''), type, value, value_type, min_invoices, is_active, start_date, end_date, created_at, updated_at
		FROM rewards %s ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, where)
	args = append(args, perPage, (page-1)*perPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var rewards []model.Reward
	for rows.Next() {
		var rw model.Reward
		if err := rows.Scan(
			&rw.ID, &rw.TenantID, &rw.Name, &rw.Description,
			&rw.Type, &rw.Value, &rw.ValueType, &rw.MinInvoices,
			&rw.IsActive, &rw.StartDate, &rw.EndDate,
			&rw.CreatedAt, &rw.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		rewards = append(rewards, rw)
	}
	return rewards, total, nil
}

// Referral

func generateReferralCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (r *rewardRepository) CreateReferral(ctx context.Context, ref *model.Referral) error {
	ref.ID = id.New()
	ref.CreatedAt = time.Now()
	if ref.ReferralCode == "" {
		ref.ReferralCode = generateReferralCode()
	}
	if ref.Status == "" {
		ref.Status = "pending"
	}

	query := `
		INSERT INTO referrals (id, tenant_id, referrer_id, referred_id, reward_id, referral_code, status, reward_amount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.Exec(ctx, query,
		ref.ID, ref.TenantID, ref.ReferrerID, ref.ReferredID,
		ref.RewardID, ref.ReferralCode, ref.Status, ref.RewardAmount,
		ref.CreatedAt,
	)
	return err
}

func (r *rewardRepository) GetReferral(ctx context.Context, tenantID, referralID string) (*model.Referral, error) {
	query := `
		SELECT rf.id, rf.tenant_id, rf.referrer_id, rf.referred_id, rf.reward_id, rf.referral_code,
		       rf.status, rf.reward_amount, rf.qualified_at, rf.rewarded_at, rf.created_at,
		       COALESCE(c1.name, '') as referrer_name, COALESCE(c2.name, '') as referred_name
		FROM referrals rf
		LEFT JOIN customers c1 ON c1.id = rf.referrer_id AND c1.tenant_id = $2
		LEFT JOIN customers c2 ON c2.id = rf.referred_id AND c2.tenant_id = $2
		WHERE rf.id = $1 AND rf.tenant_id = $2
	`
	var ref model.Referral
	err := r.db.QueryRow(ctx, query, referralID, tenantID).Scan(
		&ref.ID, &ref.TenantID, &ref.ReferrerID, &ref.ReferredID,
		&ref.RewardID, &ref.ReferralCode, &ref.Status, &ref.RewardAmount,
		&ref.QualifiedAt, &ref.RewardedAt, &ref.CreatedAt,
		&ref.ReferrerName, &ref.ReferredName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

func (r *rewardRepository) GetReferralByCode(ctx context.Context, tenantID, code string) (*model.Referral, error) {
	query := `
		SELECT rf.id, rf.tenant_id, rf.referrer_id, rf.referred_id, rf.reward_id, rf.referral_code,
		       rf.status, rf.reward_amount, rf.qualified_at, rf.rewarded_at, rf.created_at,
		       COALESCE(c1.name, '') as referrer_name, COALESCE(c2.name, '') as referred_name
		FROM referrals rf
		LEFT JOIN customers c1 ON c1.id = rf.referrer_id AND c1.tenant_id = $2
		LEFT JOIN customers c2 ON c2.id = rf.referred_id AND c2.tenant_id = $2
		WHERE rf.referral_code = $1 AND rf.tenant_id = $2
	`
	var ref model.Referral
	err := r.db.QueryRow(ctx, query, code, tenantID).Scan(
		&ref.ID, &ref.TenantID, &ref.ReferrerID, &ref.ReferredID,
		&ref.RewardID, &ref.ReferralCode, &ref.Status, &ref.RewardAmount,
		&ref.QualifiedAt, &ref.RewardedAt, &ref.CreatedAt,
		&ref.ReferrerName, &ref.ReferredName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

func (r *rewardRepository) ListReferrals(ctx context.Context, tenantID string, filter ReferralFilter) ([]model.Referral, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("rf.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.CustomerID != "" {
		conditions = append(conditions, fmt.Sprintf("(rf.referrer_id = $%d OR rf.referred_id = $%d)", argIdx, argIdx))
		args = append(args, filter.CustomerID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("rf.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM referrals rf "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT rf.id, rf.tenant_id, rf.referrer_id, rf.referred_id, rf.reward_id, rf.referral_code,
		       rf.status, rf.reward_amount, rf.qualified_at, rf.rewarded_at, rf.created_at,
		       COALESCE(c1.name, '') as referrer_name, COALESCE(c2.name, '') as referred_name
		FROM referrals rf
		LEFT JOIN customers c1 ON c1.id = rf.referrer_id AND c1.tenant_id = rf.tenant_id
		LEFT JOIN customers c2 ON c2.id = rf.referred_id AND c2.tenant_id = rf.tenant_id
		%s ORDER BY rf.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var refs []model.Referral
	for rows.Next() {
		var ref model.Referral
		if err := rows.Scan(
			&ref.ID, &ref.TenantID, &ref.ReferrerID, &ref.ReferredID,
			&ref.RewardID, &ref.ReferralCode, &ref.Status, &ref.RewardAmount,
			&ref.QualifiedAt, &ref.RewardedAt, &ref.CreatedAt,
			&ref.ReferrerName, &ref.ReferredName,
		); err != nil {
			return nil, 0, err
		}
		refs = append(refs, ref)
	}
	return refs, total, nil
}

func (r *rewardRepository) UpdateReferralStatus(ctx context.Context, tenantID, referralID, status string) error {
	query := `UPDATE referrals SET status = $1 WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.Exec(ctx, query, status, referralID, tenantID)
	return err
}

func (r *rewardRepository) MarkRewarded(ctx context.Context, tenantID, referralID string) error {
	now := time.Now()
	query := `UPDATE referrals SET status = 'rewarded', rewarded_at = $1 WHERE id = $2 AND tenant_id = $3`
	_, err := r.db.Exec(ctx, query, now, referralID, tenantID)
	return err
}

func (r *rewardRepository) GetCustomerReferralCode(ctx context.Context, tenantID, customerID string) (string, error) {
	var code string
	err := r.db.QueryRow(ctx,
		`SELECT referral_code FROM referrals WHERE tenant_id = $1 AND referrer_id = $2 ORDER BY created_at DESC LIMIT 1`,
		tenantID, customerID,
	).Scan(&code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return code, nil
}

// Reward Claims

func (r *rewardRepository) CreateClaim(ctx context.Context, claim *model.RewardClaim) error {
	claim.ID = id.New()
	claim.CreatedAt = time.Now()
	if claim.Status == "" {
		claim.Status = "pending"
	}

	query := `
		INSERT INTO reward_claims (id, tenant_id, customer_id, reward_id, referral_id, amount, type, status, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		claim.ID, claim.TenantID, claim.CustomerID, claim.RewardID,
		claim.ReferralID, claim.Amount, claim.Type, claim.Status,
		claim.ExpiresAt, claim.CreatedAt,
	)
	return err
}

func (r *rewardRepository) ListClaims(ctx context.Context, tenantID string, filter ClaimFilter) ([]model.RewardClaim, int, error) {
	if filter.PerPage <= 0 {
		filter.PerPage = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("rc.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.CustomerID != "" {
		conditions = append(conditions, fmt.Sprintf("rc.customer_id = $%d", argIdx))
		args = append(args, filter.CustomerID)
		argIdx++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("rc.status = $%d", argIdx))
		args = append(args, filter.Status)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM reward_claims rc "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT rc.id, rc.tenant_id, rc.customer_id, rc.reward_id, rc.referral_id, rc.amount, rc.type,
		       rc.status, rc.applied_at, rc.expires_at, rc.created_at,
		       COALESCE(c.name, '') as customer_name, COALESCE(rw.name, '') as reward_name
		FROM reward_claims rc
		LEFT JOIN customers c  ON c.id  = rc.customer_id AND c.tenant_id  = rc.tenant_id
		LEFT JOIN rewards   rw ON rw.id = rc.reward_id   AND rw.tenant_id = rc.tenant_id
		%s ORDER BY rc.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, filter.PerPage, (filter.Page-1)*filter.PerPage)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var claims []model.RewardClaim
	for rows.Next() {
		var cl model.RewardClaim
		if err := rows.Scan(
			&cl.ID, &cl.TenantID, &cl.CustomerID, &cl.RewardID,
			&cl.ReferralID, &cl.Amount, &cl.Type, &cl.Status,
			&cl.AppliedAt, &cl.ExpiresAt, &cl.CreatedAt,
			&cl.CustomerName, &cl.RewardName,
		); err != nil {
			return nil, 0, err
		}
		claims = append(claims, cl)
	}
	return claims, total, nil
}

func (r *rewardRepository) ApplyClaim(ctx context.Context, tenantID, claimID string) error {
	now := time.Now()
	query := `UPDATE reward_claims SET status = 'applied', applied_at = $1 WHERE id = $2 AND tenant_id = $3 AND status = 'pending'`
	result, err := r.db.Exec(ctx, query, now, claimID, tenantID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("claim not found or already applied")
	}
	return nil
}

func (r *rewardRepository) GetCustomerBalance(ctx context.Context, tenantID, customerID string) (int64, error) {
	var balance int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM reward_claims WHERE tenant_id = $1 AND customer_id = $2 AND status = 'pending'`,
		tenantID, customerID,
	).Scan(&balance)
	return balance, err
}

func (r *rewardRepository) GetRewardStats(ctx context.Context, tenantID string) (*RewardStats, error) {
	var s RewardStats

	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM referrals WHERE tenant_id = $1) as total_referrals,
			(SELECT COUNT(*) FROM referrals WHERE tenant_id = $1 AND status IN ('qualified', 'rewarded')) as qualified_referrals,
			(SELECT COALESCE(SUM(amount), 0) FROM reward_claims WHERE tenant_id = $1 AND status = 'applied') as total_rewarded,
			(SELECT COUNT(*) FROM reward_claims WHERE tenant_id = $1 AND status = 'pending') as pending_claims,
			(SELECT COUNT(*) FROM rewards WHERE tenant_id = $1 AND is_active = true) as active_rewards
	`, tenantID).Scan(
		&s.TotalReferrals, &s.QualifiedReferrals, &s.TotalRewarded,
		&s.PendingClaims, &s.ActiveRewards,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *rewardRepository) FindPendingReferralByReferred(ctx context.Context, tenantID, referredID string) (*model.Referral, error) {
	query := `
		SELECT rf.id, rf.tenant_id, rf.referrer_id, rf.referred_id, rf.reward_id, rf.referral_code,
		       rf.status, rf.reward_amount, rf.qualified_at, rf.rewarded_at, rf.created_at,
		       COALESCE(c1.name, '') as referrer_name, COALESCE(c2.name, '') as referred_name
		FROM referrals rf
		LEFT JOIN customers c1 ON c1.id = rf.referrer_id AND c1.tenant_id = $1
		LEFT JOIN customers c2 ON c2.id = rf.referred_id AND c2.tenant_id = $1
		WHERE rf.tenant_id = $1 AND rf.referred_id = $2 AND rf.status = 'pending'
		LIMIT 1
	`
	var ref model.Referral
	err := r.db.QueryRow(ctx, query, tenantID, referredID).Scan(
		&ref.ID, &ref.TenantID, &ref.ReferrerID, &ref.ReferredID,
		&ref.RewardID, &ref.ReferralCode, &ref.Status, &ref.RewardAmount,
		&ref.QualifiedAt, &ref.RewardedAt, &ref.CreatedAt,
		&ref.ReferrerName, &ref.ReferredName,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ref, nil
}

func (r *rewardRepository) QualifyReferral(ctx context.Context, tenantID, referralID string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE referrals SET status = 'qualified', qualified_at = $1 WHERE id = $2 AND tenant_id = $3 AND status = 'pending'`,
		now, referralID, tenantID,
	)
	return err
}

func (r *rewardRepository) FindActiveRewardByType(ctx context.Context, tenantID, rewardType string) (*model.Reward, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(description,''), type, value, value_type, min_invoices, is_active, start_date, end_date, created_at, updated_at
		FROM rewards
		WHERE tenant_id = $1 AND type = $2 AND is_active = true
		  AND (start_date IS NULL OR start_date <= NOW())
		  AND (end_date IS NULL OR end_date >= NOW())
		ORDER BY created_at DESC
		LIMIT 1
	`
	var rw model.Reward
	err := r.db.QueryRow(ctx, query, tenantID, rewardType).Scan(
		&rw.ID, &rw.TenantID, &rw.Name, &rw.Description,
		&rw.Type, &rw.Value, &rw.ValueType, &rw.MinInvoices,
		&rw.IsActive, &rw.StartDate, &rw.EndDate,
		&rw.CreatedAt, &rw.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rw, nil
}

func (r *rewardRepository) FindActiveLoyaltyRewards(ctx context.Context, tenantID string) ([]model.Reward, error) {
	query := `
		SELECT id, tenant_id, name, COALESCE(description,''), type, value, value_type, min_invoices, is_active, start_date, end_date, created_at, updated_at
		FROM rewards
		WHERE tenant_id = $1 AND type = 'loyalty' AND is_active = true
		  AND (start_date IS NULL OR start_date <= NOW())
		  AND (end_date IS NULL OR end_date >= NOW())
		ORDER BY min_invoices ASC
	`
	rows, err := r.db.Query(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rewards []model.Reward
	for rows.Next() {
		var rw model.Reward
		if err := rows.Scan(
			&rw.ID, &rw.TenantID, &rw.Name, &rw.Description,
			&rw.Type, &rw.Value, &rw.ValueType, &rw.MinInvoices,
			&rw.IsActive, &rw.StartDate, &rw.EndDate,
			&rw.CreatedAt, &rw.UpdatedAt,
		); err != nil {
			return nil, err
		}
		rewards = append(rewards, rw)
	}
	return rewards, nil
}

func (r *rewardRepository) HasLoyaltyClaim(ctx context.Context, tenantID, customerID, rewardID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM reward_claims WHERE tenant_id = $1 AND customer_id = $2 AND reward_id = $3`,
		tenantID, customerID, rewardID,
	).Scan(&count)
	return count > 0, err
}

func (r *rewardRepository) ExpireOldClaims(ctx context.Context) (int, error) {
	result, err := r.db.Exec(ctx,
		`UPDATE reward_claims SET status = 'expired' WHERE status = 'pending' AND expires_at IS NOT NULL AND expires_at < NOW()`,
	)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (r *rewardRepository) GetRewardDashboard(ctx context.Context, tenantID string, months int) (*RewardDashboard, error) {
	if months <= 0 {
		months = 12
	}
	d := &RewardDashboard{}

	// 1. Summary stats (reuse existing query)
	stats, err := r.GetRewardStats(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("reward stats: %w", err)
	}
	d.Stats = *stats

	// 2. Monthly trends (referrals + claims) for last N months
	trendRows, err := r.db.Query(ctx, `
		WITH months AS (
			SELECT
				EXTRACT(YEAR  FROM gs)::int AS year,
				EXTRACT(MONTH FROM gs)::int AS month
			FROM generate_series(
				date_trunc('month', NOW()) - ($2 - 1) * interval '1 month',
				date_trunc('month', NOW()),
				interval '1 month'
			) AS gs
		),
		ref_data AS (
			SELECT
				EXTRACT(YEAR  FROM created_at)::int AS year,
				EXTRACT(MONTH FROM created_at)::int AS month,
				COUNT(*)                             AS total_referrals,
				COUNT(*) FILTER (WHERE status IN ('qualified','rewarded')) AS qualified
			FROM referrals
			WHERE tenant_id = $1
			GROUP BY 1, 2
		),
		claim_data AS (
			SELECT
				EXTRACT(YEAR  FROM created_at)::int AS year,
				EXTRACT(MONTH FROM created_at)::int AS month,
				COUNT(*)                             AS total_claims,
				COALESCE(SUM(amount) FILTER (WHERE status = 'applied'), 0) AS claimed_amount
			FROM reward_claims
			WHERE tenant_id = $1
			GROUP BY 1, 2
		)
		SELECT
			m.year, m.month,
			COALESCE(r.total_referrals, 0),
			COALESCE(r.qualified, 0),
			COALESCE(c.total_claims, 0),
			COALESCE(c.claimed_amount, 0)
		FROM months m
		LEFT JOIN ref_data   r ON r.year = m.year AND r.month = m.month
		LEFT JOIN claim_data c ON c.year = m.year AND c.month = m.month
		ORDER BY m.year, m.month
	`, tenantID, months)
	if err != nil {
		return nil, fmt.Errorf("monthly trends: %w", err)
	}
	defer trendRows.Close()
	for trendRows.Next() {
		var t RewardMonthlyTrend
		if err := trendRows.Scan(&t.Year, &t.Month, &t.Referrals, &t.Qualified, &t.Claims, &t.ClaimedAmount); err != nil {
			return nil, err
		}
		d.MonthlyTrends = append(d.MonthlyTrends, t)
	}

	// 3. Top referrers (top 10)
	topRows, err := r.db.Query(ctx, `
		SELECT
			rf.referrer_id,
			COALESCE(c.name, '') AS customer_name,
			COUNT(*)             AS referral_count,
			COUNT(*) FILTER (WHERE rf.status IN ('qualified','rewarded')) AS qualified_count,
			COALESCE(SUM(rc.amount) FILTER (WHERE rc.status = 'applied'), 0) AS total_rewarded
		FROM referrals rf
		LEFT JOIN customers  c  ON c.id = rf.referrer_id AND c.tenant_id = $1
		LEFT JOIN reward_claims rc ON rc.referral_id = rf.id AND rc.tenant_id = $1
		WHERE rf.tenant_id = $1
		GROUP BY rf.referrer_id, c.name
		ORDER BY qualified_count DESC, referral_count DESC
		LIMIT 10
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("top referrers: %w", err)
	}
	defer topRows.Close()
	for topRows.Next() {
		var t TopReferrer
		if err := topRows.Scan(&t.CustomerID, &t.CustomerName, &t.ReferralCount, &t.QualifiedCount, &t.TotalRewarded); err != nil {
			return nil, err
		}
		d.TopReferrers = append(d.TopReferrers, t)
	}

	// 4. Recent referrals (last 10)
	recentRefRows, err := r.db.Query(ctx, `
		SELECT rf.id, rf.tenant_id, rf.referrer_id, rf.referred_id, rf.reward_id, rf.referral_code,
		       rf.status, rf.reward_amount, rf.qualified_at, rf.rewarded_at, rf.created_at,
		       COALESCE(c1.name, '') AS referrer_name, COALESCE(c2.name, '') AS referred_name
		FROM referrals rf
		LEFT JOIN customers c1 ON c1.id = rf.referrer_id AND c1.tenant_id = $1
		LEFT JOIN customers c2 ON c2.id = rf.referred_id AND c2.tenant_id = $1
		WHERE rf.tenant_id = $1
		ORDER BY rf.created_at DESC
		LIMIT 10
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("recent referrals: %w", err)
	}
	defer recentRefRows.Close()
	for recentRefRows.Next() {
		var ref model.Referral
		if err := recentRefRows.Scan(
			&ref.ID, &ref.TenantID, &ref.ReferrerID, &ref.ReferredID,
			&ref.RewardID, &ref.ReferralCode, &ref.Status, &ref.RewardAmount,
			&ref.QualifiedAt, &ref.RewardedAt, &ref.CreatedAt,
			&ref.ReferrerName, &ref.ReferredName,
		); err != nil {
			return nil, err
		}
		d.RecentReferrals = append(d.RecentReferrals, ref)
	}

	// 5. Recent claims (last 10)
	recentClaimRows, err := r.db.Query(ctx, `
		SELECT rc.id, rc.tenant_id, rc.customer_id, rc.reward_id, rc.referral_id,
		       rc.amount, rc.type, rc.status, rc.applied_at, rc.expires_at, rc.created_at,
		       COALESCE(c.name, '') AS customer_name, COALESCE(rw.name, '') AS reward_name
		FROM reward_claims rc
		LEFT JOIN customers c  ON c.id  = rc.customer_id AND c.tenant_id = $1
		LEFT JOIN rewards   rw ON rw.id = rc.reward_id  AND rw.tenant_id = $1
		WHERE rc.tenant_id = $1
		ORDER BY rc.created_at DESC
		LIMIT 10
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("recent claims: %w", err)
	}
	defer recentClaimRows.Close()
	for recentClaimRows.Next() {
		var cl model.RewardClaim
		if err := recentClaimRows.Scan(
			&cl.ID, &cl.TenantID, &cl.CustomerID, &cl.RewardID,
			&cl.ReferralID, &cl.Amount, &cl.Type, &cl.Status,
			&cl.AppliedAt, &cl.ExpiresAt, &cl.CreatedAt,
			&cl.CustomerName, &cl.RewardName,
		); err != nil {
			return nil, err
		}
		d.RecentClaims = append(d.RecentClaims, cl)
	}

	// 6. Claim breakdown by type + status
	breakRows, err := r.db.Query(ctx, `
		SELECT type, status, COUNT(*), COALESCE(SUM(amount), 0)
		FROM reward_claims
		WHERE tenant_id = $1
		GROUP BY type, status
		ORDER BY type, status
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("claim breakdown: %w", err)
	}
	defer breakRows.Close()
	for breakRows.Next() {
		var s ClaimTypeStat
		if err := breakRows.Scan(&s.Type, &s.Status, &s.Count, &s.TotalAmount); err != nil {
			return nil, err
		}
		d.ClaimBreakdown = append(d.ClaimBreakdown, s)
	}

	return d, nil
}
