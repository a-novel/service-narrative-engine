UPDATE scenes
SET
  title = ?2,
  description = ?3,
  prompt = ?4,
  updated_at = ?5
WHERE
  project_id = ?0
  AND id = ?1
RETURNING
  *;
