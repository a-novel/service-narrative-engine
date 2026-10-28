-- A position past the last layer appends. The layers at or after the insert position move down to make
-- room. The positions unique constraint is deferred, so the shift and the insert run in one statement.
WITH
  start AS (
    SELECT
      LEAST(?1, count(*))::integer AS position
    FROM
      layers
    WHERE
      project_id = ?0
  ),
  shifted AS (
    UPDATE layers
    SET
      position = layers.position + cardinality(ARRAY[?2]::uuid[])
    FROM
      start
    WHERE
      layers.project_id = ?0
      AND layers.position >= start.position
  )
INSERT INTO
  layers (
    id,
    project_id,
    position,
    kind,
    name,
    description,
    updated_at
  )
SELECT
  data.id,
  ?0,
  start.position + (data.position - 1)::integer,
  data.kind,
  data.name,
  data.description,
  ?6
FROM
  unnest(
    ARRAY[?2]::uuid[],
    ARRAY[?3]::text[],
    ARRAY[?4]::text[],
    ARRAY[?5]::text[]
  ) WITH ORDINALITY AS data (id, kind, name, description, position),
  start
RETURNING
  *;
