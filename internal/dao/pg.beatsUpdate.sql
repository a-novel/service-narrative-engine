-- The update runs only when the project holds every beat it names, so it updates all of them or none.
WITH
  data AS (
    SELECT
      *
    FROM
      unnest(
        ARRAY[?1]::uuid[],
        ARRAY[?2]::text[],
        ARRAY[?3]::text[],
        ARRAY[?4]::text[],
        ARRAY[?5]::text[]
      ) AS data (id, title, color, description, prompt)
  ),
  held AS (
    SELECT
      count(*) AS count
    FROM
      beats
      JOIN data ON data.id = beats.id
    WHERE
      beats.project_id = ?0
  )
UPDATE beats
SET
  title = data.title,
  color = data.color,
  description = data.description,
  prompt = data.prompt,
  updated_at = ?6
FROM
  data,
  held
WHERE
  beats.project_id = ?0
  AND beats.id = data.id
  AND held.count = (
    SELECT
      count(*)
    FROM
      data
  )
RETURNING
  beats.*;
