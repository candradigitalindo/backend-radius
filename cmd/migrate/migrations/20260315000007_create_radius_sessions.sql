-- migrate:up
CREATE TABLE radius_sessions (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) REFERENCES customers(id) ON DELETE SET NULL,
    router_id       CHAR(26) REFERENCES routers(id) ON DELETE SET NULL,
    session_id      VARCHAR(100) NOT NULL,
    username        VARCHAR(100) NOT NULL,
    nas_ip_address  VARCHAR(45) NOT NULL,
    framed_ip       VARCHAR(45),
    caller_id       VARCHAR(50),
    input_octets    BIGINT DEFAULT 0,
    output_octets   BIGINT DEFAULT 0,
    started_at      TIMESTAMPTZ NOT NULL,
    updated_at      TIMESTAMPTZ,
    ended_at        TIMESTAMPTZ,
    session_time    INTEGER DEFAULT 0,
    status          VARCHAR(20) DEFAULT 'active',
    terminate_cause VARCHAR(50)
);

CREATE INDEX idx_radius_sessions_tenant ON radius_sessions(tenant_id);
CREATE INDEX idx_radius_sessions_username ON radius_sessions(username);
CREATE INDEX idx_radius_sessions_session_id ON radius_sessions(session_id);
CREATE INDEX idx_radius_sessions_status ON radius_sessions(status);
CREATE INDEX idx_radius_sessions_started ON radius_sessions(started_at);

-- migrate:down
DROP TABLE IF EXISTS radius_sessions;
