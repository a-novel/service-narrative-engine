UPDATE scenes
SET
  title = ?2,
  color = ?3,
  description = ?4,
  prompt = ?5,
  updated_at = ?6
FROM
  projects
WHERE
  scenes.id = ?0
  AND projects.id = scenes.project_id
  AND projects.owner_id = ?1
RETURNING
  scenes.*;
