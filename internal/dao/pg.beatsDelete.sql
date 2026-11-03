-- Each beat after a deleted one moves up by the number of deleted beats before it, so positions stay
-- dense. The update still sees the rows the delete removes, so it skips them by ID. The project's stored
-- bytes lose the bytes of the beats and of the scenes and cells their delete cascades to, which the shared
-- snapshot still shows.
WITH
  deleted AS (
    DELETE FROM beats
    WHERE
      project_id = ?0
      AND id = ANY (ARRAY[?1]::uuid[])
    RETURNING
      *
  ),
  shifted AS (
    UPDATE beats
    SET
      position = beats.position - (
        SELECT
          count(*)
        FROM
          deleted
        WHERE
          deleted.position < beats.position
      )::integer
    WHERE
      beats.project_id = ?0
      AND beats.id <> ALL (ARRAY[?1]::uuid[])
      AND beats.position > (
        SELECT
          min(position)
        FROM
          deleted
      )
  ),
  removed AS (
    SELECT
      project_id,
      bytes
    FROM
      deleted
    UNION ALL
    SELECT
      scenes.project_id,
      scenes.bytes
    FROM
      scenes
      JOIN deleted ON deleted.project_id = scenes.project_id
      AND deleted.id = scenes.beat_id
    UNION ALL
    SELECT
      cells.project_id,
      cells.bytes
    FROM
      cells
      JOIN scenes ON scenes.id = cells.scene_id
      JOIN deleted ON deleted.project_id = scenes.project_id
      AND deleted.id = scenes.beat_id
  ),
  -- Grouping by project yields no row when the delete matches none, which leaves the count as it is.
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes - change.bytes
    FROM
      (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          removed
        GROUP BY
          project_id
      ) AS change
    WHERE
      projects.id = change.project_id
  )
SELECT
  *
FROM
  deleted;
