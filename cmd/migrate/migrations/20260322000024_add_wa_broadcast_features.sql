-- migrate:up

ALTER TABLE broadcasts ADD COLUMN IF NOT EXISTS image_url TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS wa_broadcast_templates (
    id          VARCHAR(26) PRIMARY KEY,
    tenant_id   VARCHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(30) NOT NULL DEFAULT 'pengumuman',
    message     TEXT NOT NULL DEFAULT '',
    image_url   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wa_broadcast_templates_tenant ON wa_broadcast_templates(tenant_id);

-- migrate:down

DROP TABLE IF EXISTS wa_broadcast_templates;
ALTER TABLE broadcasts DROP COLUMN IF EXISTS image_url;
