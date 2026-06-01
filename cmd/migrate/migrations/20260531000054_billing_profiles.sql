-- migrate:up

CREATE TABLE billing_profiles (
    id             TEXT PRIMARY KEY,
    tenant_id      TEXT NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    description    TEXT NOT NULL DEFAULT '',

    -- Model: 'cycle' = tanggal invoice & JT tetap setiap bulan (level tenant/profil)
    --        'fixed' = tanggal JT = tanggal join_date customer (per customer)
    billing_model  TEXT NOT NULL DEFAULT 'cycle'
                   CHECK (billing_model IN ('cycle', 'fixed')),

    -- Untuk model 'cycle': invoice_day = tanggal generate invoice, due_day = tanggal JT
    -- Untuk model 'fixed': due_day tidak dipakai (JT = hari join_date customer)
    --                      invoice_day = berapa hari sebelum JT invoice digenerate (default 7)
    invoice_day    INT NOT NULL DEFAULT 0,  -- 0 = auto (due_day - 7)
    due_day        INT NOT NULL DEFAULT 1,  -- 1-28

    -- Isolir & grace (berlaku untuk kedua model)
    isolir_day     INT NOT NULL DEFAULT 3,   -- hari setelah JT sebelum isolir (0 = langsung isolir)
    grace_period   INT NOT NULL DEFAULT 0,   -- hari toleransi setelah isolir

    -- Kapan bayar: 'due_date' = bayar saat/sebelum JT
    --             'advance'  = bayar di muka saat aktivasi (invoice langsung dibuat)
    payment_timing TEXT NOT NULL DEFAULT 'due_date'
                   CHECK (payment_timing IN ('due_date', 'advance')),

    is_default     BOOLEAN NOT NULL DEFAULT FALSE,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_billing_profiles_tenant ON billing_profiles(tenant_id);
CREATE INDEX idx_billing_profiles_default ON billing_profiles(tenant_id, is_default) WHERE is_default = TRUE;

-- Tambah referensi billing_profile_id ke customers
ALTER TABLE customers ADD COLUMN IF NOT EXISTS billing_profile_id TEXT REFERENCES billing_profiles(id) ON DELETE SET NULL;
CREATE INDEX idx_customers_billing_profile ON customers(billing_profile_id) WHERE billing_profile_id IS NOT NULL;

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS billing_profile_id;
DROP TABLE IF EXISTS billing_profiles;
