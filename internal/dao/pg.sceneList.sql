SELECT
  scenes.*,
  count(*) OVER () AS total
FROM
  scenes
  JOIN beats ON beats.id = scenes.beat_id
WHERE
  scenes.project_id = ?0
ORDER BY
  beats.position,
  scenes.position
LIMIT
  ?1
OFFSET
  ?2;
