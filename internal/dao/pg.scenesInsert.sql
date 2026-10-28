-- A position past the last scene appends. The scenes at or after the insert position move down to make
-- room. The positions unique constraint is deferred, so the shift and the insert run in one statement.
WITH
  start AS (
    SELECT
      LEAST(?1, count(*))::integer AS position
    FROM
      scenes
    WHERE
      project_id = ?0
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position + cardinality(ARRAY[?2]::uuid[])
    FROM
      start
    WHERE
      scenes.project_id = ?0
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
  data.id,
  ?0,
  start.position + (data.position - 1)::integer,
  data.title,
  data.color,
  data.description,
  data.prompt,
  ?7
FROM
  unnest(
    ARRAY[?2]::uuid[],
    ARRAY[?3]::text[],
    ARRAY[?4]::text[],
    ARRAY[?5]::text[],
    ARRAY[?6]::text[]
  ) WITH ORDINALITY AS data (id, title, color, description, prompt, position),
  start
RETURNING
  *;
