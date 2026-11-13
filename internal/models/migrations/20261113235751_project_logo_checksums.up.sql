-- A logo's checksum is the SHA-256 of its bytes, so a client compares the logos it holds with the stored ones
-- without reading any bytes. A stored column hashes a logo once, when it is written; a virtual one, PostgreSQL 18's
-- default, would hash the bytes on every read. Adding the column hashes every stored logo and rewrites the table
-- under an ACCESS EXCLUSIVE lock, which every project read waits on. On 2026-11-14 production's whole database held
-- 8.1 MiB, 0.6 MiB above an empty one (cnpg_pg_database_size_bytes), so the rewrite holds the lock for milliseconds
-- and a plain statement fits it. A table of hundreds of MiB would need a nullable column backfilled in batches.
ALTER TABLE project_logos
ADD COLUMN checksum bytea GENERATED ALWAYS AS (sha256(data)) STORED;
