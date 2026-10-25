SELECT
  layers.*
FROM
  layers
  JOIN projects ON projects.id = layers.project_id
WHERE
  layers.id = ?0
  AND projects.owner_id = ?1;
