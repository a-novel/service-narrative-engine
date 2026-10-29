-- A position past the last scene of the beat appends. The beat's scenes at or after the insert position
-- move down to make room. The positions unique constraint is deferred, so the shift and the insert run in
-- one statement.
WITH
  start AS (
    SELECT
      LEAST(?2, count(*))::integer AS position
    FROM
      scenes
    WHERE
      project_id = ?0
      AND beat_id = ?1
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position + cardinality(ARRAY[?3]::uuid[])
    FROM
      start
    WHERE
      scenes.project_id = ?0
      AND scenes.beat_id = ?1
      AND scenes.position >= start.position
  )
INSERT INTO
  scenes (
    id,
    project_id,
    beat_id,
    position,
    title,
    description,
    prompt,
    updated_at
  )
SELECT
  data.id,
  ?0,
  ?1,
  start.position + (data.position - 1)::integer,
  data.title,
  data.description,
  data.prompt,
  ?7
FROM
  unnest(
    ARRAY[?3]::uuid[],
    ARRAY[?4]::text[],
    ARRAY[?5]::text[],
    ARRAY[?6]::text[]
  ) WITH ORDINALITY AS data (id, title, description, prompt, position),
  start
RETURNING
  *;
