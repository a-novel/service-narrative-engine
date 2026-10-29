-- The length counts the characters of each cell's stored JSON document.
SELECT
  scene_id,
  sum(length(content::text))::integer AS length
FROM
  cells
WHERE
  project_id = ?0
  AND scene_id = ANY (ARRAY[?1]::uuid[])
GROUP BY
  scene_id
LIMIT
  ?2;
