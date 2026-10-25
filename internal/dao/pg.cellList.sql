SELECT
  cells.*
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
WHERE
  scenes.project_id = ?0
ORDER BY
  scenes.position,
  cells.layer_id
LIMIT
  ?1;
