-- One statement reads either module, so a read loads the module it asks for and never the other. A version
-- that ships without the module, as most do without a migration, returns no row.
SELECT
  *
FROM
  (
    SELECT
      CASE ?5
        WHEN 'ui' THEN ui
        WHEN 'migration' THEN migration
      END AS code,
      encode(
        CASE ?5
          WHEN 'ui' THEN ui_checksum
          WHEN 'migration' THEN migration_checksum
        END,
        'hex'
      ) AS checksum,
      published_at
    FROM
      plugin_versions
    WHERE
      plugin = ?0
      AND major = ?1
      AND minor = ?2
      AND patch = ?3
      AND prerelease = ?4
  ) AS module
WHERE
  code IS NOT NULL;
