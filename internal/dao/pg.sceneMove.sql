-- A position past the last scene moves it last. The scenes between the old and the new position move by one
-- toward the old position, so positions stay dense. The positions unique constraint is deferred, so they
-- all change in one statement.
WITH
  moving AS (
    SELECT
      position
    FROM
      scenes
    WHERE
      project_id = ?0
      AND id = ?1
  ),
  target AS (
    SELECT
      LEAST(?2, count(*) - 1)::integer AS position
    FROM
      scenes
    WHERE
      project_id = ?0
  ),
  moved AS (
    UPDATE scenes
    SET
      position = CASE
        WHEN scenes.id = ?1 THEN target.position
        WHEN target.position < moving.position THEN scenes.position + 1
        ELSE scenes.position - 1
      END
    FROM
      moving,
      target
    WHERE
      scenes.project_id = ?0
      AND scenes.position BETWEEN LEAST(moving.position, target.position) AND GREATEST(moving.position, target.position)
    RETURNING
      scenes.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
