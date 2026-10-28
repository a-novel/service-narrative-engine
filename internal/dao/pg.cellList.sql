-- A cell pairs one scene and one layer of the project. The position checks cap a project at 500
-- scenes and 20 layers, so this limit returns every cell.
SELECT
  cells.*
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
  JOIN layers ON layers.id = cells.layer_id
WHERE
  scenes.project_id = ?0
  AND layers.project_id = ?0
ORDER BY
  scenes.position,
  cells.layer_id
LIMIT
  10000;
