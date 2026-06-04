-- migrate:up
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS status           VARCHAR(20)  DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS fingerprint      VARCHAR(64)  DEFAULT '',
    ADD COLUMN IF NOT EXISTS registration_ip  VARCHAR(45)  DEFAULT '',
    ADD COLUMN IF NOT EXISTS risk_score       INTEGER      DEFAULT 0,
    ADD COLUMN IF NOT EXISTS risk_reasons     TEXT         DEFAULT '';

-- Set existing rows to active
UPDATE tenants SET status = 'active' WHERE status IS NULL OR status = '';

-- migrate:down
ALTER TABLE tenants
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS fingerprint,
    DROP COLUMN IF EXISTS registration_ip,
    DROP COLUMN IF EXISTS risk_score,
    DROP COLUMN IF EXISTS risk_reasons;
