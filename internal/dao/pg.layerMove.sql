-- The layers between the old and the new position move by one toward the old position, so positions
-- stay dense. The positions unique constraint is deferred, so they all change in one statement.
WITH
  moving AS (
    SELECT
      project_id,
      position
    FROM
      layers
    WHERE
      id = ?0
  ),
  moved AS (
    UPDATE layers
    SET
      position = CASE
        WHEN layers.id = ?0 THEN ?1
        WHEN ?1 < moving.position THEN layers.position + 1
        ELSE layers.position - 1
      END
    FROM
      moving
    WHERE
      layers.project_id = moving.project_id
      AND layers.position BETWEEN LEAST(moving.position, ?1) AND GREATEST(moving.position, ?1)
    RETURNING
      layers.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?0;
