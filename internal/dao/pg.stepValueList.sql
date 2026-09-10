SELECT
  id,
  project_id,
  key,
  value,
  created_at
FROM
  step_values
WHERE
  project_id = ?0
  AND key = ?1
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  ?2;
