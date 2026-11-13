SELECT
  id,
  edited_at
FROM
  projects
WHERE
  owner_id = ?0
LIMIT
  ?1;
