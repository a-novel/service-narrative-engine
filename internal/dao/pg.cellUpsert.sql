-- The update applies to a cell of the same project only. A scene and a layer of another project that
-- already form a cell match no row, so the write returns none.
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
WHERE
  cells.project_id = EXCLUDED.project_id
RETURNING
  scene_id,
  layer_id,
  updated_at;
