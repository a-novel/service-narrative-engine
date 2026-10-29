-- Locking in ID order keeps two writes that share scenes from waiting on each other in a cycle.
SELECT
  1
FROM
  scenes
WHERE
  project_id = ?0
  AND id = ANY (ARRAY[?1]::uuid[])
ORDER BY
  id
LIMIT
  ?2
FOR UPDATE;
