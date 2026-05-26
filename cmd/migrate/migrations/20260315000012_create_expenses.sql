-- migrate:up
CREATE TABLE expense_categories (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name            VARCHAR(50) NOT NULL,
    color           VARCHAR(7) DEFAULT '#6B7280',
    UNIQUE(tenant_id, name)
);

CREATE TABLE expenses (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id     CHAR(26) REFERENCES expense_categories(id),
    description     VARCHAR(200) NOT NULL,
    amount          BIGINT NOT NULL,
    expense_date    DATE NOT NULL DEFAULT CURRENT_DATE,
    receipt_url     VARCHAR(255),
    created_by      CHAR(26) NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_expenses_tenant ON expenses(tenant_id);
CREATE INDEX idx_expenses_date ON expenses(tenant_id, expense_date);

-- migrate:down
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS expense_categories;
