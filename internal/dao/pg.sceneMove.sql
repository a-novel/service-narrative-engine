-- The scenes between the old and the new position move by one toward the old position, so positions
-- stay dense. The positions unique constraint is deferred, so they all change in one statement.
WITH
  moving AS (
    SELECT
      project_id,
      position
    FROM
      scenes
    WHERE
      id = ?0
  ),
  moved AS (
    UPDATE scenes
    SET
      position = CASE
        WHEN scenes.id = ?0 THEN ?1
        WHEN ?1 < moving.position THEN scenes.position + 1
        ELSE scenes.position - 1
      END
    FROM
      moving
    WHERE
      scenes.project_id = moving.project_id
      AND scenes.position BETWEEN LEAST(moving.position, ?1) AND GREATEST(moving.position, ?1)
    RETURNING
      scenes.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?0;
