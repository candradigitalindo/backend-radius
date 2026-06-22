-- migrate:up
CREATE TABLE IF NOT EXISTS router_iface_samples (
    id         BIGSERIAL PRIMARY KEY,
    router_id  VARCHAR(40)  NOT NULL,
    tenant_id  VARCHAR(40)  NOT NULL,
    iface      VARCHAR(64)  NOT NULL,
    rx_bytes   BIGINT       NOT NULL,            -- raw interface counter (bytes)
    tx_bytes   BIGINT       NOT NULL,
    rx_bps     BIGINT       NOT NULL DEFAULT 0,  -- computed throughput (bit/s)
    tx_bps     BIGINT       NOT NULL DEFAULT 0,
    sampled_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_iface_samples_lookup ON router_iface_samples (router_id, iface, sampled_at DESC);
CREATE INDEX IF NOT EXISTS idx_iface_samples_prune ON router_iface_samples (sampled_at);

-- migrate:down
DROP TABLE IF EXISTS router_iface_samples;
