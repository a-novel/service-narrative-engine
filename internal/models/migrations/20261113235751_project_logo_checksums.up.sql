-- A logo's checksum is the SHA-256 of its bytes, so a client compares the logos it holds with the stored ones
-- without reading any bytes. A stored column hashes a logo once, when it is written; a virtual one, PostgreSQL 18's
-- default, would hash the bytes on every read. Adding the column hashes every stored logo and rewrites the table
-- under an exclusive lock. The table holds one logo of 256 KiB at most per project, so a plain statement fits it.
ALTER TABLE project_logos
ADD COLUMN checksum bytea GENERATED ALWAYS AS (sha256(data)) STORED;
