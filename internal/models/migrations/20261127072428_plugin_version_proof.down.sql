-- Dropping the columns loses the proof of every published version, with the ID of the key that signed it. The
-- check that keeps them together goes with them.
ALTER TABLE plugin_versions
DROP COLUMN IF EXISTS api_key_id,
DROP COLUMN IF EXISTS proof;
