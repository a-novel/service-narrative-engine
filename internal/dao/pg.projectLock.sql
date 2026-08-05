SELECT
  id
FROM
  projects
WHERE
  id = ?0
  AND owner_id = ?1
FOR UPDATE;
