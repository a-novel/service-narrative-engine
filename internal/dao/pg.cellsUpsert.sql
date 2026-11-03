-- A pair that already forms a cell of another project stops the whole batch, which then writes no row.
-- The update also applies to a cell of the same project only. The project's stored bytes take the
-- difference between the cells' new bytes and their old ones, none for a new cell.
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
  ),
  written AS (
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
      project_id,
      scene_id,
      layer_id,
      updated_at,
      new.bytes - coalesce(old.bytes, 0) AS delta
  ),
  -- Grouping by project yields no row when the batch writes none, which leaves the count as it is.
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + change.delta
    FROM
      (
        SELECT
          project_id,
          sum(delta) AS delta
        FROM
          written
        GROUP BY
          project_id
      ) AS change
    WHERE
      projects.id = change.project_id
  )
SELECT
  scene_id,
  layer_id,
  updated_at
FROM
  written;
