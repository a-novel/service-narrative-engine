SELECT
  id,
  project_id,
  value,
  created_at
FROM
  manuscripts
WHERE
  project_id = ?0
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  ?1;
