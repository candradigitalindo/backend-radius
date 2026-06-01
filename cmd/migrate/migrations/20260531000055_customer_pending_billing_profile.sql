-- migrate:up

-- Menyimpan billing profile baru yang menunggu berlaku di periode berikutnya.
-- Diset ketika admin mengubah model billing saat customer masih memiliki invoice
-- periode berjalan yang belum dibayar.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS pending_billing_profile_id TEXT REFERENCES billing_profiles(id) ON DELETE SET NULL;

-- Menyimpan harga paket sebelum diubah, untuk keperluan kalkulasi prorata.
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS previous_package_price INT8 DEFAULT 0;

-- Tanggal efektif perubahan paket (digunakan untuk menghitung sisa hari prorata).
ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS package_changed_at TIMESTAMPTZ;

-- migrate:down
ALTER TABLE customers DROP COLUMN IF EXISTS pending_billing_profile_id;
ALTER TABLE customers DROP COLUMN IF EXISTS previous_package_price;
ALTER TABLE customers DROP COLUMN IF EXISTS package_changed_at;
