SELECT
  *
FROM
  projects
WHERE
  owner = ?0
ORDER BY
  created_at DESC
LIMIT
  ?1
OFFSET
  ?2;
