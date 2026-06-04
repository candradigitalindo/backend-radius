-- migrate:up
-- Allow subscription_orders to survive tenant deletion (for billing history)
ALTER TABLE subscription_orders
    ADD COLUMN IF NOT EXISTS tenant_name TEXT,
    ADD COLUMN IF NOT EXISTS tenant_slug TEXT;

-- Backfill existing rows from tenants table
UPDATE subscription_orders so
SET
    tenant_name = t.name,
    tenant_slug = t.slug
FROM tenants t
WHERE so.tenant_id = t.id
  AND so.tenant_name IS NULL;

-- Drop the old NOT NULL FK, make it nullable with ON DELETE SET NULL
ALTER TABLE subscription_orders
    ALTER COLUMN tenant_id DROP NOT NULL;

ALTER TABLE subscription_orders
    DROP CONSTRAINT IF EXISTS subscription_orders_tenant_id_fkey;

ALTER TABLE subscription_orders
    ADD CONSTRAINT subscription_orders_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE SET NULL;

-- migrate:down
ALTER TABLE subscription_orders
    DROP CONSTRAINT IF EXISTS subscription_orders_tenant_id_fkey;

ALTER TABLE subscription_orders
    ADD CONSTRAINT subscription_orders_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id);

ALTER TABLE subscription_orders
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE subscription_orders
    DROP COLUMN IF EXISTS tenant_name,
    DROP COLUMN IF EXISTS tenant_slug;
