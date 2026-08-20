-- migrate:up
-- Nomor line keluaran ODC yang dipakai ODP ini (1..N sesuai rasio ODC, mis. 1:4 → line 1-4).
-- Nullable: ODP estafet (tanpa ODC) atau data lama tidak punya line.
ALTER TABLE odps ADD COLUMN splitter_line SMALLINT;

-- Satu line ODC hanya boleh dipakai satu ODP.
CREATE UNIQUE INDEX idx_odps_splitter_line ON odps (splitter_id, splitter_line)
    WHERE splitter_id IS NOT NULL AND splitter_line IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_odps_splitter_line;
ALTER TABLE odps DROP COLUMN IF EXISTS splitter_line;
