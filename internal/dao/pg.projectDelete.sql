DELETE FROM projects
WHERE
  id = ?0
  AND owner_id = ?1
  AND revision = ?2
RETURNING
  *;
