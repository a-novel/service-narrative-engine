SELECT
  plugin,
  major,
  minor,
  patch,
  prerelease,
  migration IS NOT NULL AS has_migration,
  encode(schema_checksum, 'hex') AS schema_checksum,
  count(*) OVER () AS total
FROM
  plugin_versions
WHERE
  plugin = ?0
  AND prerelease = ''
  AND (
    (major, minor, patch) > (?1, ?2, ?3)
    -- A release comes after its own prereleases.
    OR (
      (major, minor, patch) = (?1, ?2, ?3)
      AND ?4 <> ''
    )
  )
ORDER BY
  major,
  minor,
  patch
LIMIT
  ?5
OFFSET
  ?6;
