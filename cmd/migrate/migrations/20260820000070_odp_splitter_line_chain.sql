-- migrate:up
-- Satu line ODC boleh berisi RANTAI beberapa ODP (estafet dengan splitter
-- asimetris, mis. tap 10% -> 20% -> 50% -> sisa). Keunikan pindah ke
-- (splitter, line, sequence): satu posisi urutan dalam satu line = satu ODP.
DROP INDEX IF EXISTS idx_odps_splitter_line;
CREATE UNIQUE INDEX idx_odps_splitter_line_seq ON odps (splitter_id, splitter_line, sequence)
    WHERE splitter_id IS NOT NULL AND splitter_line IS NOT NULL;

-- migrate:down
DROP INDEX IF EXISTS idx_odps_splitter_line_seq;
CREATE UNIQUE INDEX idx_odps_splitter_line ON odps (splitter_id, splitter_line)
    WHERE splitter_id IS NOT NULL AND splitter_line IS NOT NULL;
