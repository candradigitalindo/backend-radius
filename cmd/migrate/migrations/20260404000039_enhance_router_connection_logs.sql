-- migrate:up
ALTER TABLE router_connection_logs
    ADD COLUMN IF NOT EXISTS router_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS identity VARCHAR(100),
    ADD COLUMN IF NOT EXISTS router_os_ver VARCHAR(50),
    ADD COLUMN IF NOT EXISTS board_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS uptime VARCHAR(50),
    ADD COLUMN IF NOT EXISTS cpu_load INTEGER,
    ADD COLUMN IF NOT EXISTS free_memory BIGINT,
    ADD COLUMN IF NOT EXISTS total_memory BIGINT,
    ADD COLUMN IF NOT EXISTS duration VARCHAR(50);

-- migrate:down
ALTER TABLE router_connection_logs
    DROP COLUMN IF EXISTS router_name,
    DROP COLUMN IF EXISTS identity,
    DROP COLUMN IF EXISTS router_os_ver,
    DROP COLUMN IF EXISTS board_name,
    DROP COLUMN IF EXISTS uptime,
    DROP COLUMN IF EXISTS cpu_load,
    DROP COLUMN IF EXISTS free_memory,
    DROP COLUMN IF EXISTS total_memory,
    DROP COLUMN IF EXISTS duration;
