-- The position check caps a project at 500 scenes, so this limit returns every one.
SELECT
  *
FROM
  scenes
WHERE
  project_id = ?0
ORDER BY
  position
LIMIT
  500;
