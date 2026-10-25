-- The request lists the scenes to create, so they bound the rows returned.
INSERT INTO
  scenes (
    id,
    project_id,
    position,
    title,
    synopsis,
    updated_at
  )
SELECT
  data.id,
  ?0,
  ?1 + (data.position - 1)::integer,
  data.title,
  data.synopsis,
  ?5
FROM
  unnest(
    ARRAY[?2]::uuid[],
    ARRAY[?3]::text[],
    ARRAY[?4]::text[]
  ) WITH ORDINALITY AS data (id, title, synopsis, position)
RETURNING
  *;
