INSERT INTO
  cells (
    project_id,
    scene_id,
    layer_id,
    content,
    updated_at
  )
SELECT
  ?0,
  data.scene_id,
  data.layer_id,
  data.content::jsonb,
  ?4
FROM
  unnest(
    ARRAY[?1]::uuid[],
    ARRAY[?2]::uuid[],
    ARRAY[?3]::text[]
  ) AS data (scene_id, layer_id, content)
ON CONFLICT (scene_id, layer_id) DO UPDATE
SET
  content = EXCLUDED.content,
  updated_at = EXCLUDED.updated_at
RETURNING
  scene_id,
  layer_id,
  updated_at;
