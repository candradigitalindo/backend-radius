-- migrate:up
CREATE TABLE push_devices (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    device_type     VARCHAR(20) NOT NULL DEFAULT 'android',
    fcm_token       TEXT NOT NULL,
    is_active       BOOLEAN DEFAULT true,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_push_devices_tenant ON push_devices(tenant_id);
CREATE INDEX idx_push_devices_customer ON push_devices(tenant_id, customer_id);
CREATE UNIQUE INDEX idx_push_devices_unique ON push_devices(tenant_id, customer_id, fcm_token);

CREATE TABLE notifications (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    title           VARCHAR(200) NOT NULL,
    body            TEXT NOT NULL,
    data            JSONB,
    is_read         BOOLEAN DEFAULT false,
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_notifications_customer ON notifications(tenant_id, customer_id);
CREATE INDEX idx_notifications_unread ON notifications(tenant_id, customer_id, is_read) WHERE is_read = false;

-- migrate:down
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS push_devices;
