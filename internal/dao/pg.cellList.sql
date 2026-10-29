SELECT
  cells.*,
  count(*) OVER () AS total
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
  JOIN beats ON beats.id = scenes.beat_id
  JOIN layers ON layers.id = cells.layer_id
WHERE
  cells.project_id = ?0
ORDER BY
  beats.position,
  scenes.position,
  layers.position
LIMIT
  ?1
OFFSET
  ?2;
