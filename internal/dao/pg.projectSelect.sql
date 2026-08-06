SELECT
  id,
  owner_id,
  created_at
FROM
  projects
WHERE
  id = ?0
  AND owner_id = ?1;
