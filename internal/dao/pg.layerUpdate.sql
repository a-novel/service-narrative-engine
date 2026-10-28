UPDATE layers
SET
  name = ?1,
  description = ?2,
  updated_at = ?3
WHERE
  id = ?0
RETURNING
  *;
