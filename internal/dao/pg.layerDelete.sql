-- The layers after the deleted one move up by one, so positions stay dense.
WITH
  deleted AS (
    DELETE FROM layers
    WHERE
      id = ?0
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
  )
SELECT
  *
FROM
  deleted;
