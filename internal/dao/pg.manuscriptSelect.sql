-- A beat that holds no scene keeps one row with a null scene, and a scene the layer holds no cell for
-- keeps its row with a null content. The position checks cap a project at 200 beats of 10 scenes, so
-- the limit reads every row.
SELECT
  beats.id AS beat_id,
  beats.title AS beat_title,
  beats.description AS beat_description,
  scenes.title AS scene_title,
  cells.content
FROM
  beats
  LEFT JOIN scenes ON scenes.project_id = beats.project_id
  AND scenes.beat_id = beats.id
  LEFT JOIN cells ON cells.project_id = scenes.project_id
  AND cells.scene_id = scenes.id
  AND cells.layer_id = ?1
WHERE
  beats.project_id = ?0
ORDER BY
  beats.position,
  scenes.position
LIMIT
  2000;
