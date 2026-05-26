-- migrate:up

-- Update all subscription plans: all features open, no router limits, only customer count differs
UPDATE subscription_plans SET
    description = 'Cocok untuk coba semua fitur tanpa batas waktu. Upgrade kapan saja.',
    max_customers = 20,
    max_routers = 99999,
    features = '["Semua Fitur Terbuka", "Maks. 20 Pelanggan Aktif"]'::jsonb
WHERE slug = 'gratis';

UPDATE subscription_plans SET
    description = 'Untuk RT/RW Net yang baru memulai dengan fitur lengkap.',
    price = 99000,
    max_customers = 100,
    max_routers = 99999,
    features = '["Semua Fitur Terbuka", "Maks. 100 Pelanggan Aktif"]'::jsonb
WHERE slug = 'starter';

UPDATE subscription_plans SET
    description = 'Pilihan terbaik untuk RT/RW Net yang sedang berkembang.',
    price = 199000,
    max_customers = 300,
    max_routers = 99999,
    features = '["Semua Fitur Terbuka", "Maks. 300 Pelanggan Aktif"]'::jsonb,
    is_popular = TRUE
WHERE slug = 'growth';

UPDATE subscription_plans SET
    description = 'Untuk ISP skala menengah dengan banyak pelanggan.',
    price = 399000,
    max_customers = 1000,
    max_routers = 99999,
    features = '["Semua Fitur Terbuka", "Maks. 1.000 Pelanggan Aktif"]'::jsonb
WHERE slug = 'professional';

UPDATE subscription_plans SET
    description = 'Untuk ISP besar tanpa batasan pelanggan.',
    price = 699000,
    max_customers = 99999,
    max_routers = 99999,
    features = '["Semua Fitur Terbuka", "Pelanggan Unlimited"]'::jsonb
WHERE slug = 'enterprise';

-- Also update default max_routers for any future plans
ALTER TABLE subscription_plans ALTER COLUMN max_routers SET DEFAULT 99999;

-- migrate:down

-- Revert to original values
UPDATE subscription_plans SET
    description = 'Cocok untuk memulai dan mencoba semua fitur dasar.',
    max_customers = 20,
    max_routers = 1,
    features = '["Dashboard & Monitoring", "Manajemen 20 Pelanggan", "1 Router MikroTik", "Invoice & Pembayaran Manual", "Halaman Isolir", "Tiket Support"]'::jsonb
WHERE slug = 'gratis';

UPDATE subscription_plans SET
    description = 'Ideal untuk RT/RW Net pemula dengan pelanggan terbatas.',
    price = 99000,
    max_customers = 50,
    max_routers = 2,
    features = '["Semua fitur Gratis", "Manajemen 50 Pelanggan", "2 Router MikroTik", "WhatsApp Notifikasi", "Payment Gateway", "Auto Invoice & Isolir", "Laporan Keuangan"]'::jsonb
WHERE slug = 'starter';

UPDATE subscription_plans SET
    description = 'Untuk RT/RW Net yang sedang berkembang pesat.',
    price = 199000,
    max_customers = 200,
    max_routers = 5,
    features = '["Semua fitur Starter", "Manajemen 200 Pelanggan", "5 Router MikroTik", "FTTH / OLT Management", "ODP & ONT Monitoring", "Voucher Hotspot", "Bandwidth Monitoring", "GenieACS TR-069", "Reseller Management"]'::jsonb,
    is_popular = TRUE
WHERE slug = 'growth';

UPDATE subscription_plans SET
    description = 'Solusi lengkap untuk ISP skala menengah.',
    price = 399000,
    max_customers = 1000,
    max_routers = 20,
    features = '["Semua fitur Growth", "Manajemen 1.000 Pelanggan", "20 Router MikroTik", "IPAM (IP Address Management)", "Reward & Referral Program", "Broadcast WhatsApp Massal", "Reminder Tagihan Otomatis", "Custom Halaman Isolir", "Export Laporan Excel"]'::jsonb
WHERE slug = 'professional';

UPDATE subscription_plans SET
    description = 'Untuk ISP besar tanpa batasan.',
    price = 699000,
    max_customers = 99999,
    max_routers = 99999,
    features = '["Semua fitur Professional", "Pelanggan Unlimited", "Router Unlimited", "Priority Support", "Custom Domain", "API Access"]'::jsonb
WHERE slug = 'enterprise';

ALTER TABLE subscription_plans ALTER COLUMN max_routers SET DEFAULT 1;
