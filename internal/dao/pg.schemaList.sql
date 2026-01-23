SELECT
  *
FROM
  (
    SELECT DISTINCT
      ON (module_id, module_namespace) *
    FROM
      schemas
    WHERE
      project_id = ?0
      AND data IS NOT NULL
    ORDER BY
      module_id,
      module_namespace,
      created_at DESC
  ) AS latest_schemas
ORDER BY
  created_at;
