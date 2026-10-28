-- The scenes between the old and the new position move by one toward the old position, so positions
-- stay dense. The positions unique constraint is deferred, so they all change in one statement.
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
  moved AS (
    UPDATE scenes
    SET
      position = CASE
        WHEN scenes.id = ?1 THEN ?2
        WHEN ?2 < moving.position THEN scenes.position + 1
        ELSE scenes.position - 1
      END
    FROM
      moving
    WHERE
      scenes.project_id = ?0
      AND scenes.position BETWEEN LEAST(moving.position, ?2) AND GREATEST(moving.position, ?2)
    RETURNING
      scenes.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
