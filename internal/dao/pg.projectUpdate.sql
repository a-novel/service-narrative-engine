UPDATE projects
SET
  title = ?3,
  premise = ?4,
  language = ?5,
  revision = revision + 1,
  updated_at = ?6
WHERE
  id = ?0
  AND owner_id = ?1
  AND revision = ?2
RETURNING
  *;
