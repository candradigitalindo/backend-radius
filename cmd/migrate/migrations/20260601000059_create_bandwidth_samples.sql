-- migrate:up
CREATE TABLE IF NOT EXISTS bandwidth_samples (
    id           CHAR(26)    PRIMARY KEY,
    tenant_id    CHAR(26)    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id  CHAR(26)    REFERENCES customers(id) ON DELETE SET NULL,
    session_id   VARCHAR(64) NOT NULL,
    sampled_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    interval_sec INTEGER     NOT NULL DEFAULT 0,
    download_bps BIGINT      NOT NULL DEFAULT 0,
    upload_bps   BIGINT      NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_bw_samples_tenant_sampled  ON bandwidth_samples(tenant_id, sampled_at DESC);
CREATE INDEX IF NOT EXISTS idx_bw_samples_customer        ON bandwidth_samples(customer_id);
CREATE INDEX IF NOT EXISTS idx_bw_samples_session         ON bandwidth_samples(session_id);

-- migrate:down
DROP TABLE IF EXISTS bandwidth_samples;
