-- Each layer after a deleted one moves up by the number of deleted layers before it, so positions stay
-- dense. The update still sees the rows the delete removes, so it skips them by ID.
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
  )
SELECT
  *
FROM
  deleted;
