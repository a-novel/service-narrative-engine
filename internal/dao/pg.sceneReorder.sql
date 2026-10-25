-- The caller lists every scene of the project, so IDs bound the rows returned. The positions
-- unique constraint is deferred, so every position changes in one statement.
UPDATE scenes
SET
  position = (data.position - 1)::integer
FROM
  unnest(ARRAY[?1]::uuid[]) WITH ORDINALITY AS data (id, position)
WHERE
  scenes.project_id = ?0
  AND scenes.id = data.id
RETURNING
  scenes.*;
