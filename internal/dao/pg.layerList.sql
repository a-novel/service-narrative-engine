-- The position check caps a project at 1000 layers, so this limit returns every one.
SELECT
  *
FROM
  layers
WHERE
  project_id = ?0
ORDER BY
  position
LIMIT
  1000;
