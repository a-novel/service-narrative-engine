-- A position past the last beat appends. The beats at or after the insert position move down by one
-- to make room. The positions unique constraint is deferred, so the shift and the insert run in one
-- statement.
WITH
  start AS (
    SELECT
      LEAST(?2, count(*))::integer AS position
    FROM
      beats
    WHERE
      project_id = ?1
  ),
  shifted AS (
    UPDATE beats
    SET
      position = beats.position + 1
    FROM
      start
    WHERE
      beats.project_id = ?1
      AND beats.position >= start.position
  )
INSERT INTO
  beats (
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
