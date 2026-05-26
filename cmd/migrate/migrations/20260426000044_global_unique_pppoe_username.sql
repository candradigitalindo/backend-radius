-- Make pppoe_username globally unique (across all tenants)
-- Drop old per-tenant unique constraint
ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_tenant_id_pppoe_username_key;

-- Add global unique constraint (only for non-empty values)
CREATE UNIQUE INDEX IF NOT EXISTS customers_pppoe_username_global_key
    ON customers (pppoe_username)
    WHERE pppoe_username IS NOT NULL AND pppoe_username != '';
