-- migrate:up
CREATE TABLE roles (
    id          CHAR(26) PRIMARY KEY,
    tenant_id   CHAR(26) NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    slug        VARCHAR(50) NOT NULL,
    description TEXT DEFAULT '',
    is_system   BOOLEAN DEFAULT FALSE,
    permissions TEXT[] DEFAULT '{}',
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE INDEX idx_roles_tenant ON roles(tenant_id);
CREATE INDEX idx_roles_tenant_slug ON roles(tenant_id, slug);

-- migrate:down
DROP TABLE IF EXISTS roles;
