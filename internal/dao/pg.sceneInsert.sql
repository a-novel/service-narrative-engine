-- A position past the last scene appends. The scenes at or after the insert position move down by one
-- to make room. The positions unique constraint is deferred, so the shift and the insert run in one
-- statement.
WITH
  start AS (
    SELECT
      LEAST(?2, count(*))::integer AS position
    FROM
      scenes
    WHERE
      project_id = ?1
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position + 1
    FROM
      start
    WHERE
      scenes.project_id = ?1
      AND scenes.position >= start.position
  )
INSERT INTO
  scenes (
    id,
    project_id,
    position,
    title,
    color,
    description,
    prompt,
    updated_at
  )
SELECT
  ?0,
  ?1,
  start.position,
  ?3,
  ?4,
  ?5,
  ?6,
  ?7
FROM
  start
RETURNING
  *;
