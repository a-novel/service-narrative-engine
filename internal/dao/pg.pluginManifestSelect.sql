-- The manifest leaves the modules out, so reading it loads neither.
SELECT
  plugin,
  major,
  minor,
  patch,
  prerelease,
  schema,
  exports,
  migration IS NOT NULL AS has_migration,
  encode(schema_checksum, 'hex') AS schema_checksum,
  encode(ui_checksum, 'hex') AS ui_checksum,
  proof,
  api_key_id,
  published_at
FROM
  plugin_versions
WHERE
  plugin = ?0
  AND major = ?1
  AND minor = ?2
  AND patch = ?3
  AND prerelease = ?4;
