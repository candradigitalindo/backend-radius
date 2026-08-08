-- migrate:up
-- onts.customer_id was created without an ON DELETE action, unlike every other
-- customer_id foreign key in the schema. This blocks deleting any customer that
-- has an ONT assigned (i.e. FTTH customers) with a plain FK violation, which the
-- API surfaces as a generic "gagal menghapus pelanggan" error. Unassign the ONT
-- instead of blocking the delete, matching radius_sessions/ipam behavior.
ALTER TABLE onts DROP CONSTRAINT IF EXISTS onts_customer_id_fkey;
ALTER TABLE onts ADD CONSTRAINT onts_customer_id_fkey
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL;

-- migrate:down
ALTER TABLE onts DROP CONSTRAINT IF EXISTS onts_customer_id_fkey;
ALTER TABLE onts ADD CONSTRAINT onts_customer_id_fkey
    FOREIGN KEY (customer_id) REFERENCES customers(id);
