UPDATE layers
SET
  name = ?3,
  description = ?4,
  revision = layers.revision + 1,
  updated_at = ?5
FROM
  projects
WHERE
  layers.id = ?0
  AND projects.id = layers.project_id
  AND projects.owner_id = ?1
  AND layers.revision = ?2
RETURNING
  layers.*;
