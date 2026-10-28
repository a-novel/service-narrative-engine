-- A cell's key never changes, so a page that starts after the last key of the previous one skips
-- and repeats nothing while scenes and layers move. The primary key serves this order.
SELECT
  cells.*
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
WHERE
  scenes.project_id = ?0
  AND (cells.scene_id, cells.layer_id) > (?2, ?3)
ORDER BY
  cells.scene_id,
  cells.layer_id
LIMIT
  ?1;
