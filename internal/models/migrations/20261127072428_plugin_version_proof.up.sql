-- A published version carries the proof of who published it: the token the publisher's API key signed for this
-- version, which a client verifies before it uses the version. The client reads that key from JSON Keys by its ID,
-- stored beside the proof. Both are null for a version published on a local stack whose publisher check is open,
-- and the table constraint keeps them together. Only the release job publishes, a row per plugin release, so the
-- registry stays small and a plain statement checks it within the ALTER's lock.
ALTER TABLE plugin_versions
ADD COLUMN proof text CHECK (proof <> ''),
ADD COLUMN api_key_id text CHECK (api_key_id <> ''),
ADD CONSTRAINT plugin_versions_proof_api_key_id_check CHECK ((proof IS NULL) = (api_key_id IS NULL));
