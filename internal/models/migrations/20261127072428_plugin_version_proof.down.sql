-- Dropping the columns loses the proof of every published version, with the ID of the key that signed it.
ALTER TABLE plugin_versions
DROP CONSTRAINT IF EXISTS plugin_versions_proof_api_key_id_check,
DROP COLUMN IF EXISTS api_key_id,
DROP COLUMN IF EXISTS proof;
