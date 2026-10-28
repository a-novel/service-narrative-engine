DELETE FROM projects
WHERE
  id = ?0
RETURNING
  *;
