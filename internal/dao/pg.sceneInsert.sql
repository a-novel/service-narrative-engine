-- The scenes at or after the insert position move down to make room. The positions unique constraint
-- is deferred, so the shift and the insert run in one statement.
WITH
  shifted AS (
    UPDATE scenes
    SET
      position = position + cardinality(ARRAY[?2]::uuid[])
    WHERE
      project_id = ?0
      AND position >= ?1
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
  ?1 + (data.position - 1)::integer,
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
  ) WITH ORDINALITY AS data (id, title, color, description, prompt, position)
RETURNING
  *;
