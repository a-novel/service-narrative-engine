-- The layers between the old and the new position move by one toward the old position, so positions
-- stay dense. The positions unique constraint is deferred, so they all change in one statement.
WITH
  moving AS (
    SELECT
      position
    FROM
      layers
    WHERE
      project_id = ?0
      AND id = ?1
  ),
  moved AS (
    UPDATE layers
    SET
      position = CASE
        WHEN layers.id = ?1 THEN ?2
        WHEN ?2 < moving.position THEN layers.position + 1
        ELSE layers.position - 1
      END
    FROM
      moving
    WHERE
      layers.project_id = ?0
      AND layers.position BETWEEN LEAST(moving.position, ?2) AND GREATEST(moving.position, ?2)
    RETURNING
      layers.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
