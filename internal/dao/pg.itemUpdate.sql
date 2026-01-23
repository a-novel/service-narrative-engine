UPDATE items
SET
  name = ?0,
  description = ?1,
  updated_at = ?2
WHERE
  id = ?3
RETURNING
  *;
