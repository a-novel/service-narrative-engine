-- A pair that already forms a cell of another project stops the whole batch, which then writes no row.
-- The update also applies to a cell of the same project only.
WITH
  data AS (
    SELECT
      *
    FROM
      unnest(
        ARRAY[?1]::uuid[],
        ARRAY[?2]::uuid[],
        ARRAY[?3]::text[]
      ) AS data (scene_id, layer_id, content)
  ),
  foreign_cells AS (
    SELECT
      1
    FROM
      cells
      JOIN data ON data.scene_id = cells.scene_id
      AND data.layer_id = cells.layer_id
    WHERE
      cells.project_id <> ?0
    LIMIT
      1
  )
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
  data
WHERE
  NOT EXISTS (
    SELECT
      1
    FROM
      foreign_cells
  )
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
