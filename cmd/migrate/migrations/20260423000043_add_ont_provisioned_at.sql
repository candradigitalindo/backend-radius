-- Add provisioned_at to track whether GenieACS TR-069 provisioning has been completed.
-- NULL = belum terprovisi (device belum online saat pendaftaran, perlu retry worker).
-- NOT NULL = sudah berhasil diprovisi.
ALTER TABLE onts ADD COLUMN IF NOT EXISTS provisioned_at TIMESTAMPTZ DEFAULT NULL;
