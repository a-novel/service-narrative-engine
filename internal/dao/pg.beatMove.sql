-- A position past the last beat moves it last. The beats between the old and the new position move by one
-- toward the old position, so positions stay dense. The positions unique constraint is deferred, so they
-- all change in one statement.
WITH
  moving AS (
    SELECT
      position
    FROM
      beats
    WHERE
      project_id = ?0
      AND id = ?1
  ),
  target AS (
    SELECT
      LEAST(?2, count(*) - 1)::integer AS position
    FROM
      beats
    WHERE
      project_id = ?0
  ),
  moved AS (
    UPDATE beats
    SET
      position = CASE
        WHEN beats.id = ?1 THEN target.position
        WHEN target.position < moving.position THEN beats.position + 1
        ELSE beats.position - 1
      END
    FROM
      moving,
      target
    WHERE
      beats.project_id = ?0
      AND beats.position BETWEEN LEAST(moving.position, target.position) AND GREATEST(moving.position, target.position)
    RETURNING
      beats.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
