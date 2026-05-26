-- migrate:up

-- Remove per-customer cost and Priority Support from features text
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 100 Pelanggan Aktif"]'::jsonb WHERE slug = 'starter';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 300 Pelanggan Aktif"]'::jsonb WHERE slug = 'growth';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 1.000 Pelanggan Aktif"]'::jsonb WHERE slug = 'professional';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Pelanggan Unlimited"]'::jsonb WHERE slug = 'enterprise';

-- migrate:down

UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 100 Pelanggan Aktif", "~Rp 990/pelanggan/bulan"]'::jsonb WHERE slug = 'starter';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 300 Pelanggan Aktif", "~Rp 663/pelanggan/bulan"]'::jsonb WHERE slug = 'growth';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Maks. 1.000 Pelanggan Aktif", "~Rp 399/pelanggan/bulan"]'::jsonb WHERE slug = 'professional';
UPDATE subscription_plans SET features = '["Semua Fitur Terbuka", "Pelanggan Unlimited", "Priority Support"]'::jsonb WHERE slug = 'enterprise';
