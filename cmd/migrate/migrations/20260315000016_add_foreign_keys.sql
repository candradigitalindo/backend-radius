-- migrate:up
ALTER TABLE odp_ports
    ADD CONSTRAINT fk_odp_ports_customer
    FOREIGN KEY (customer_id) REFERENCES customers(id) ON DELETE SET NULL;

-- migrate:down
ALTER TABLE odp_ports DROP CONSTRAINT IF EXISTS fk_odp_ports_customer;
