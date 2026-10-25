-- The caller lists every layer of the project, so IDs bound the rows returned. The positions
-- unique constraint is deferred, so every position changes in one statement.
UPDATE layers
SET
  position = (data.position - 1)::integer
FROM
  unnest(ARRAY[?1]::uuid[]) WITH ORDINALITY AS data (id, position)
WHERE
  layers.project_id = ?0
  AND layers.id = data.id
RETURNING
  layers.*;
