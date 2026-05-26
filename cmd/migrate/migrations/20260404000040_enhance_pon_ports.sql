-- migrate:up
ALTER TABLE pon_ports
    ADD COLUMN IF NOT EXISTS if_index        INT,
    ADD COLUMN IF NOT EXISTS admin_status     VARCHAR(20) DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS oper_status      VARCHAR(20) DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS in_octets        BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS out_octets       BIGINT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS sfp_vendor       VARCHAR(100),
    ADD COLUMN IF NOT EXISTS sfp_model        VARCHAR(100),
    ADD COLUMN IF NOT EXISTS sfp_rx_power     DECIMAL(8,2),
    ADD COLUMN IF NOT EXISTS sfp_tx_power     DECIMAL(8,2),
    ADD COLUMN IF NOT EXISTS sfp_temperature  DECIMAL(6,2),
    ADD COLUMN IF NOT EXISTS sfp_voltage      DECIMAL(6,2),
    ADD COLUMN IF NOT EXISTS last_polled_at   TIMESTAMPTZ;

-- Unique constraint for SNMP auto-discovery (upsert by olt_id + if_index)
CREATE UNIQUE INDEX IF NOT EXISTS idx_pon_ports_olt_ifindex ON pon_ports(olt_id, if_index) WHERE if_index IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_pon_ports_olt_ifindex;
ALTER TABLE pon_ports
    DROP COLUMN IF EXISTS if_index,
    DROP COLUMN IF EXISTS admin_status,
    DROP COLUMN IF EXISTS oper_status,
    DROP COLUMN IF EXISTS in_octets,
    DROP COLUMN IF EXISTS out_octets,
    DROP COLUMN IF EXISTS sfp_vendor,
    DROP COLUMN IF EXISTS sfp_model,
    DROP COLUMN IF EXISTS sfp_rx_power,
    DROP COLUMN IF EXISTS sfp_tx_power,
    DROP COLUMN IF EXISTS sfp_temperature,
    DROP COLUMN IF EXISTS sfp_voltage,
    DROP COLUMN IF EXISTS last_polled_at;
