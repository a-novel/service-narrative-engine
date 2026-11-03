-- The update applies to a cell of the same project only. A scene and a layer of another project that
-- already form a cell match no row, so the write returns none. The project's stored bytes take the
-- difference between the cell's new bytes and its old ones, none for a new cell.
WITH
  written AS (
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
      project_id,
      scene_id,
      layer_id,
      updated_at,
      new.bytes - coalesce(old.bytes, 0) AS delta
  ),
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + written.delta
    FROM
      written
    WHERE
      projects.id = written.project_id
  )
SELECT
  scene_id,
  layer_id,
  updated_at
FROM
  written;
