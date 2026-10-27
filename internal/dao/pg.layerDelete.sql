DELETE FROM layers USING projects
WHERE
  layers.id = ?0
  AND projects.id = layers.project_id
  AND projects.owner_id = ?1
RETURNING
  layers.*;
