-- The update runs only when the project holds every scene it names, so it updates all of them or none.
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
      scenes
      JOIN data ON data.id = scenes.id
    WHERE
      scenes.project_id = ?0
  )
UPDATE scenes
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
  scenes.project_id = ?0
  AND scenes.id = data.id
  AND held.count = (
    SELECT
      count(*)
    FROM
      data
  )
RETURNING
  scenes.*;
