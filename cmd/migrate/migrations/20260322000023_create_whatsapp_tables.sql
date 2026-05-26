-- migrate:up

CREATE TABLE IF NOT EXISTS whatsapp_configs (
    id         VARCHAR(26) PRIMARY KEY,
    tenant_id  VARCHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_url    VARCHAR(500) NOT NULL DEFAULT '',
    api_key    VARCHAR(500) NOT NULL DEFAULT '',
    sender_number VARCHAR(30) NOT NULL DEFAULT '',
    is_active  BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);

CREATE TABLE IF NOT EXISTS whatsapp_logs (
    id         VARCHAR(26) PRIMARY KEY,
    tenant_id  VARCHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    phone      VARCHAR(30) NOT NULL DEFAULT '',
    message    TEXT NOT NULL DEFAULT '',
    status     VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_msg  TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_whatsapp_logs_tenant ON whatsapp_logs(tenant_id, created_at DESC);

-- migrate:down

DROP TABLE IF EXISTS whatsapp_logs;
DROP TABLE IF EXISTS whatsapp_configs;
