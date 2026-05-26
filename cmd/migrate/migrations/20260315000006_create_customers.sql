-- migrate:up
CREATE TABLE customers (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_code   VARCHAR(20) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    phone           VARCHAR(20),
    email           VARCHAR(100),
    address         TEXT,
    latitude        DECIMAL(10, 8),
    longitude       DECIMAL(11, 8),
    connection_type VARCHAR(10) DEFAULT 'pppoe',
    pppoe_username  VARCHAR(50),
    pppoe_password  VARCHAR(50),
    ip_address      VARCHAR(15),
    package_id      CHAR(26) REFERENCES packages(id),
    router_id       CHAR(26) REFERENCES routers(id),
    odp_port_id     CHAR(26) REFERENCES odp_ports(id),
    join_date       DATE NOT NULL DEFAULT CURRENT_DATE,
    billing_date    SMALLINT DEFAULT 1,
    custom_price    BIGINT,
    discount        BIGINT DEFAULT 0,
    additional_fee  BIGINT DEFAULT 0,
    fee_description VARCHAR(100),
    status          VARCHAR(20) DEFAULT 'active',
    isolated_at     TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, customer_code),
    UNIQUE(tenant_id, pppoe_username)
);

CREATE INDEX idx_customers_tenant ON customers(tenant_id);
CREATE INDEX idx_customers_status ON customers(tenant_id, status);
CREATE INDEX idx_customers_router ON customers(router_id);
CREATE INDEX idx_customers_package ON customers(package_id);

-- migrate:down
DROP TABLE IF EXISTS customers;
