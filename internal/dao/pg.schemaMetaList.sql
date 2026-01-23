SELECT
  id,
  project_id,
  owner,
  module_id,
  module_namespace,
  module_version,
  module_preversion,
  source,
  created_at,
  FALSE AS is_nil,
  TRUE AS is_latest
FROM
  (
    SELECT DISTINCT
      ON (module_id, module_namespace) *
    FROM
      schemas
    WHERE
      project_id = ?0
    ORDER BY
      module_id,
      module_namespace,
      created_at DESC
  ) AS latest_schemas
WHERE
  data IS NOT NULL
ORDER BY
  created_at DESC;
