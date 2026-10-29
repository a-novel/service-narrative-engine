-- A scene the layer holds no cell for keeps its row, with a null content. The position check caps a
-- project at 500 scenes, so the limit reads all of them.
SELECT
  scenes.title,
  cells.content
FROM
  scenes
  LEFT JOIN cells ON cells.project_id = scenes.project_id
  AND cells.scene_id = scenes.id
  AND cells.layer_id = ?1
WHERE
  scenes.project_id = ?0
ORDER BY
  scenes.position
LIMIT
  500;
