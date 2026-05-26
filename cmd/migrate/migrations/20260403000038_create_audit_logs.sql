-- migrate:up

CREATE TABLE audit_logs (
    id          CHAR(26) PRIMARY KEY,
    user_id     CHAR(26) NOT NULL,
    user_email  VARCHAR(254) NOT NULL DEFAULT '',
    role        VARCHAR(30) NOT NULL DEFAULT '',
    tenant_id   CHAR(26),
    action      VARCHAR(50) NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    resource_id VARCHAR(100),
    method      VARCHAR(10) NOT NULL,
    path        TEXT NOT NULL,
    ip_address  VARCHAR(45) NOT NULL DEFAULT '',
    user_agent  TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    detail      TEXT,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource);

-- migrate:down

DROP TABLE IF EXISTS audit_logs;
