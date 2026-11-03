-- The layers after the deleted one move up by one, so positions stay dense. The project's stored bytes lose
-- the bytes of the layer and of the cells its delete cascades to. The parts of one WITH share a snapshot,
-- so the sum still sees those cells.
WITH
  deleted AS (
    DELETE FROM layers
    WHERE
      project_id = ?0
      AND id = ?1
    RETURNING
      *
  ),
  shifted AS (
    UPDATE layers
    SET
      position = layers.position - 1
    FROM
      deleted
    WHERE
      layers.project_id = deleted.project_id
      AND layers.position > deleted.position
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
