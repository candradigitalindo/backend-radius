-- migrate:up
-- Migration 20260513000049_reward_program intended stricter constraints on the reward
-- tables, but used CREATE TABLE IF NOT EXISTS over tables already created by
-- 20260321000022_create_rewards. The IF NOT EXISTS made those definitions a no-op, so the
-- intended CHECK constraints and NOT NULL columns were never applied. The reward tables are
-- empty in production, so we apply the missing constraints here idempotently.

-- rewards: constrain type / value_type
ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_type_check;
ALTER TABLE rewards ADD CONSTRAINT rewards_type_check CHECK (type IN ('referral','loyalty','promo'));
ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_value_type_check;
ALTER TABLE rewards ADD CONSTRAINT rewards_value_type_check CHECK (value_type IN ('fixed','percentage'));

-- referrals: required foreign keys + status check
ALTER TABLE referrals ALTER COLUMN referred_id SET NOT NULL;
ALTER TABLE referrals ALTER COLUMN reward_id SET NOT NULL;
ALTER TABLE referrals DROP CONSTRAINT IF EXISTS referrals_status_check;
ALTER TABLE referrals ADD CONSTRAINT referrals_status_check CHECK (status IN ('pending','qualified','rewarded','expired'));

-- reward_claims: constrain type / status
ALTER TABLE reward_claims DROP CONSTRAINT IF EXISTS reward_claims_type_check;
ALTER TABLE reward_claims ADD CONSTRAINT reward_claims_type_check CHECK (type IN ('invoice_discount','balance_credit'));
ALTER TABLE reward_claims DROP CONSTRAINT IF EXISTS reward_claims_status_check;
ALTER TABLE reward_claims ADD CONSTRAINT reward_claims_status_check CHECK (status IN ('pending','applied','expired'));

-- migrate:down
ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_type_check;
ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_value_type_check;
ALTER TABLE referrals ALTER COLUMN referred_id DROP NOT NULL;
ALTER TABLE referrals ALTER COLUMN reward_id DROP NOT NULL;
ALTER TABLE referrals DROP CONSTRAINT IF EXISTS referrals_status_check;
ALTER TABLE reward_claims DROP CONSTRAINT IF EXISTS reward_claims_type_check;
ALTER TABLE reward_claims DROP CONSTRAINT IF EXISTS reward_claims_status_check;
