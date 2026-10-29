-- The beats after the deleted one move up by one, so positions stay dense.
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
  )
SELECT
  *
FROM
  deleted;
