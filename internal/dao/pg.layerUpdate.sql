UPDATE layers
SET
  name = ?2,
  description = ?3,
  updated_at = ?4
WHERE
  project_id = ?0
  AND id = ?1
RETURNING
  *;
