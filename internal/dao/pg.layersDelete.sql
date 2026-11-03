-- Each layer after a deleted one moves up by the number of deleted layers before it, so positions stay
-- dense. The update still sees the rows the delete removes, so it skips them by ID. The project's stored
-- bytes lose the bytes of the layers and of the cells their delete cascades to, which the shared snapshot
-- still shows.
WITH
  deleted AS (
    DELETE FROM layers
    WHERE
      project_id = ?0
      AND id = ANY (ARRAY[?1]::uuid[])
    RETURNING
      *
  ),
  shifted AS (
    UPDATE layers
    SET
      position = layers.position - (
        SELECT
          count(*)
        FROM
          deleted
        WHERE
          deleted.position < layers.position
      )::integer
    WHERE
      layers.project_id = ?0
      AND layers.id <> ALL (ARRAY[?1]::uuid[])
      AND layers.position > (
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
      cells.project_id,
      cells.bytes
    FROM
      cells
      JOIN deleted ON deleted.project_id = cells.project_id
      AND deleted.id = cells.layer_id
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
