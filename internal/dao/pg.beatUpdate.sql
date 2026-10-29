UPDATE beats
SET
  title = ?2,
  color = ?3,
  description = ?4,
  prompt = ?5,
  updated_at = ?6
WHERE
  project_id = ?0
  AND id = ?1
RETURNING
  *;
