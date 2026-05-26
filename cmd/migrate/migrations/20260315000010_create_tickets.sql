-- migrate:up
CREATE TABLE tickets (
    id              CHAR(26) PRIMARY KEY,
    tenant_id       CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id     CHAR(26) NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    ticket_number   VARCHAR(20) NOT NULL,
    subject         VARCHAR(200) NOT NULL,
    description     TEXT,
    category        VARCHAR(30) DEFAULT 'general',
    priority        VARCHAR(10) DEFAULT 'medium',
    status          VARCHAR(20) DEFAULT 'open',
    assigned_to     CHAR(26) REFERENCES users(id),
    resolved_at     TIMESTAMPTZ,
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_tickets_tenant ON tickets(tenant_id);
CREATE INDEX idx_tickets_customer ON tickets(customer_id);
CREATE INDEX idx_tickets_status ON tickets(tenant_id, status);

CREATE TABLE ticket_messages (
    id              CHAR(26) PRIMARY KEY,
    ticket_id       CHAR(26) NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    sender_type     VARCHAR(10) NOT NULL,
    sender_id       CHAR(26) NOT NULL,
    message         TEXT NOT NULL,
    attachment_url  VARCHAR(255),
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_ticket_messages_ticket ON ticket_messages(ticket_id);

-- migrate:down
DROP TABLE IF EXISTS ticket_messages;
DROP TABLE IF EXISTS tickets;
