SELECT
  cells.*,
  count(*) OVER () AS total
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
  JOIN layers ON layers.id = cells.layer_id
WHERE
  cells.project_id = ?0
ORDER BY
  scenes.position,
  layers.position
LIMIT
  ?1
OFFSET
  ?2;
