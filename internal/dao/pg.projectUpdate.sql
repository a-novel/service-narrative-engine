UPDATE projects
SET
  title = ?1,
  workflow = ?2::text[],
  updated_at = ?3
WHERE
  id = ?0
RETURNING
  *;
