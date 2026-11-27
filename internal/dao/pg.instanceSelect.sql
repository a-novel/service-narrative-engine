-- The links of an instance are read apart, so the read of its content leaves them in their column.
SELECT
  id,
  project_id,
  plugin,
  version,
  name,
  content,
  created_at,
  updated_at
FROM
  instances
WHERE
  project_id = ?0
  AND id = ?1;
