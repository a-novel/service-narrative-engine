-- Each beat after a deleted one moves up by the number of deleted beats before it, so positions stay
-- dense. The update still sees the rows the delete removes, so it skips them by ID.
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
  )
SELECT
  *
FROM
  deleted;
