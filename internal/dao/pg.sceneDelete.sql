-- The scenes after the deleted one move up by one, so positions stay dense.
WITH
  deleted AS (
    DELETE FROM scenes
    WHERE
      id = ?0
    RETURNING
      *
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position - 1
    FROM
      deleted
    WHERE
      scenes.project_id = deleted.project_id
      AND scenes.position > deleted.position
  )
SELECT
  *
FROM
  deleted;
