UPDATE layers
SET
  name = ?2,
  description = ?3,
  updated_at = ?4
FROM
  projects
WHERE
  layers.id = ?0
  AND projects.id = layers.project_id
  AND projects.owner_id = ?1
RETURNING
  layers.*;
