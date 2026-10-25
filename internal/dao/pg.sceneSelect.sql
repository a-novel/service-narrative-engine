SELECT
  scenes.*
FROM
  scenes
  JOIN projects ON projects.id = scenes.project_id
WHERE
  scenes.id = ?0
  AND projects.owner_id = ?1;
