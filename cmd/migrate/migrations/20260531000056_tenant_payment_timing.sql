-- migrate:up
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS default_payment_timing TEXT NOT NULL DEFAULT 'due_date'
  CHECK (default_payment_timing IN ('due_date', 'advance'));

-- migrate:down
ALTER TABLE tenants DROP COLUMN IF EXISTS default_payment_timing;
