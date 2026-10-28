INSERT INTO
  cells (scene_id, layer_id, content, updated_at)
VALUES
  (?0, ?1, ?2::jsonb, ?3)
ON CONFLICT (scene_id, layer_id) DO UPDATE
SET
  content = EXCLUDED.content,
  updated_at = EXCLUDED.updated_at
RETURNING
  *;
