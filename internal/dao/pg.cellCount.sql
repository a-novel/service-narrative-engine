SELECT
  count(*) AS count
FROM
  cells
  JOIN scenes ON scenes.id = cells.scene_id
WHERE
  scenes.project_id = ?0;
