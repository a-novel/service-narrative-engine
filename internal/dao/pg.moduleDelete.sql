DELETE FROM modules
WHERE
  id = ?0
  AND namespace = ?1
  AND version = ?2
  AND preversion = ?3
RETURNING
  *;
