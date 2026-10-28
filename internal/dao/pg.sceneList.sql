SELECT
  *,
  count(*) OVER () AS total
FROM
  scenes
WHERE
  project_id = ?0
ORDER BY
  position
LIMIT
  ?1
OFFSET
  ?2;
