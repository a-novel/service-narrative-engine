-- The scene leaves its beat, whose later scenes move up by one, then joins the target beat at the target
-- position, whose scenes from there on move down by one. The target beat may be the scene's own. A position
-- past the target beat's last scene moves it last. The positions unique constraint is deferred, so every
-- position changes in one statement.
WITH
  moving AS (
    SELECT
      id,
      beat_id,
      position
    FROM
      scenes
    WHERE
      project_id = ?0
      AND id = ?1
  ),
  closed AS (
    SELECT
      scenes.id,
      scenes.beat_id,
      scenes.position - (
        scenes.beat_id = moving.beat_id
        AND scenes.position > moving.position
      )::integer AS position
    FROM
      scenes,
      moving
    WHERE
      scenes.project_id = ?0
      AND scenes.beat_id IN (moving.beat_id, ?2)
      AND scenes.id <> moving.id
  ),
  target AS (
    SELECT
      LEAST(?3, count(*))::integer AS position
    FROM
      closed
    WHERE
      beat_id = ?2
  ),
  placed AS (
    SELECT
      moving.id,
      ?2::uuid AS beat_id,
      target.position
    FROM
      moving,
      target
    UNION ALL
    SELECT
      closed.id,
      closed.beat_id,
      closed.position + (
        closed.beat_id = ?2
        AND closed.position >= target.position
      )::integer
    FROM
      closed,
      target
  ),
  moved AS (
    UPDATE scenes
    SET
      beat_id = placed.beat_id,
      position = placed.position
    FROM
      placed
    WHERE
      scenes.project_id = ?0
      AND scenes.id = placed.id
    RETURNING
      scenes.*
  )
SELECT
  *
FROM
  moved
WHERE
  id = ?1;
