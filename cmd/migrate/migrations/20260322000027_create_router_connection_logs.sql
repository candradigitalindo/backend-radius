-- migrate:up
CREATE TABLE router_connection_logs (
    id          CHAR(26) PRIMARY KEY,
    router_id   CHAR(26) NOT NULL REFERENCES routers(id) ON DELETE CASCADE,
    event       VARCHAR(20) NOT NULL,
    vpn_ip      VARCHAR(45),
    endpoint    VARCHAR(100),
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_router_conn_logs_router ON router_connection_logs(router_id, created_at DESC);

-- migrate:down
DROP TABLE IF EXISTS router_connection_logs;
