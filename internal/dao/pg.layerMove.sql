-- A position past the last layer moves it last. The layers between the old and the new position move by one
-- toward the old position, so positions stay dense. The positions unique constraint is deferred, so they
-- all change in one statement.
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
  target AS (
    SELECT
      LEAST(?2, count(*) - 1)::integer AS position
    FROM
      layers
    WHERE
      project_id = ?0
  ),
  moved AS (
    UPDATE layers
    SET
      position = CASE
        WHEN layers.id = ?1 THEN target.position
        WHEN target.position < moving.position THEN layers.position + 1
        ELSE layers.position - 1
      END
    FROM
      moving,
      target
    WHERE
      layers.project_id = ?0
      AND layers.position BETWEEN LEAST(moving.position, target.position) AND GREATEST(moving.position, target.position)
    RETURNING
      layers.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
