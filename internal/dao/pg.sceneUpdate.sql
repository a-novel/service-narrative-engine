UPDATE scenes
SET
  title = ?3,
  synopsis = ?4,
  revision = scenes.revision + 1,
  updated_at = ?5
FROM
  projects
WHERE
  scenes.id = ?0
  AND projects.id = scenes.project_id
  AND projects.owner_id = ?1
  AND scenes.revision = ?2
RETURNING
  scenes.*;
