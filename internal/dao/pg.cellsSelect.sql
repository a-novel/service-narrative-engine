-- Each key reads only what it asks for: a context that shows a cell as one version of its recap never loads
-- the content, nor the other versions.
SELECT
  cells.scene_id,
  cells.layer_id,
  CASE
    WHEN wanted.content THEN cells.content
  END AS content,
  wanted.recap AS recap_version,
  cells.recap ->> nullif(wanted.recap, '') AS recap
FROM
  unnest(
    ARRAY[?1]::uuid[],
    ARRAY[?2]::uuid[],
    ARRAY[?3]::boolean[],
    ARRAY[?4]::text[]
  ) AS wanted (scene_id, layer_id, content, recap)
  JOIN cells ON cells.project_id = ?0
  AND cells.scene_id = wanted.scene_id
  AND cells.layer_id = wanted.layer_id
LIMIT
  ?5;
