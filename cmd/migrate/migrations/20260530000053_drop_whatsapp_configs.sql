-- migrate:up
-- whatsapp_configs (api_url / api_key / sender_number / is_active) is dead: the WhatsApp
-- gateway URL and secret come from env (WA_SERVICE_URL / WA_API_SECRET), the send path never
-- reads this table, is_active gates nothing, and the frontend never calls its config endpoints.
-- whatsapp_logs is still used for send logging and is kept.
DROP TABLE IF EXISTS whatsapp_configs;

-- migrate:down
CREATE TABLE IF NOT EXISTS whatsapp_configs (
    id            VARCHAR(26) PRIMARY KEY,
    tenant_id     VARCHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_url       VARCHAR(500) NOT NULL DEFAULT '',
    api_key       VARCHAR(500) NOT NULL DEFAULT '',
    sender_number VARCHAR(30) NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id)
);
