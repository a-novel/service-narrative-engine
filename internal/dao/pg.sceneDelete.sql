DELETE FROM scenes USING projects
WHERE
  scenes.id = ?0
  AND projects.id = scenes.project_id
  AND projects.owner_id = ?1
  AND scenes.revision = ?2
RETURNING
  scenes.*;
