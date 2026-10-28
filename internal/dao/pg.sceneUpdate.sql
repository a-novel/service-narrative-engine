UPDATE scenes
SET
  title = ?1,
  color = ?2,
  description = ?3,
  prompt = ?4,
  updated_at = ?5
WHERE
  id = ?0
RETURNING
  *;
