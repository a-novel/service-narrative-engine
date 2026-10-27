UPDATE projects
SET
  title = ?2,
  premise = ?3,
  language = ?4,
  updated_at = ?5
WHERE
  id = ?0
  AND owner_id = ?1
RETURNING
  *;
