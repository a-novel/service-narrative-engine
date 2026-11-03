-- The beats after the deleted one move up by one, so positions stay dense. The project's stored bytes lose
-- the bytes of the beat and of the scenes and cells its delete cascades to. The parts of one WITH share a
-- snapshot, so the sums still see those rows.
WITH
  deleted AS (
    DELETE FROM beats
    WHERE
      project_id = ?0
      AND id = ?1
    RETURNING
      *
  ),
  shifted AS (
    UPDATE beats
    SET
      position = beats.position - 1
    FROM
      deleted
    WHERE
      beats.project_id = deleted.project_id
      AND beats.position > deleted.position
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
