UPDATE schemas
SET
  data = ?1,
  created_at = ?2
WHERE
  id = ?0
RETURNING
  *;
