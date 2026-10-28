INSERT INTO
  cells (
    project_id,
    scene_id,
    layer_id,
    content,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3::jsonb, ?4)
ON CONFLICT (scene_id, layer_id) DO UPDATE
SET
  content = EXCLUDED.content,
  updated_at = EXCLUDED.updated_at
RETURNING
  scene_id,
  layer_id,
  updated_at;
