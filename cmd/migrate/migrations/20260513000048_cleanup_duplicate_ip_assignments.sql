-- migrate:up
-- Cleanup customers who currently have more than 1 assigned IP.
-- Keep only the most recently assigned IP; release the rest back to 'available'.
WITH ranked AS (
    SELECT id, customer_id, pool_id,
           ROW_NUMBER() OVER (PARTITION BY customer_id ORDER BY assigned_at DESC NULLS LAST, updated_at DESC) AS rn
    FROM ip_addresses
    WHERE status = 'assigned' AND customer_id IS NOT NULL
),
stale AS (
    SELECT id, pool_id FROM ranked WHERE rn > 1
)
UPDATE ip_addresses
SET customer_id = NULL, status = 'available', assigned_at = NULL, updated_at = NOW()
WHERE id IN (SELECT id FROM stale);

-- Recalculate used_ips for all pools after cleanup
UPDATE ip_pools SET used_ips = (
    SELECT COUNT(*) FROM ip_addresses WHERE pool_id = ip_pools.id AND status = 'assigned'
);

-- migrate:down
-- No rollback; data is already cleaned up.
