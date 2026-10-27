-- The request lists the scenes to create, so they bound the rows returned.
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
