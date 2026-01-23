SELECT
  id,
  created_at
FROM
  schemas
WHERE
  project_id = ?0
  AND module_id = ?1
  AND module_namespace = ?2
ORDER BY
  created_at DESC
LIMIT
  ?3
OFFSET
  ?4;
