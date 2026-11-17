-- The update runs only when the project holds every layer it names, so it updates all of them or none.
WITH
  data AS (
    SELECT
      *
    FROM
      unnest(
        ARRAY[?1]::uuid[],
        ARRAY[?2]::text[],
        ARRAY[?3]::text[]
      ) AS data (id, name, description)
  ),
  held AS (
    SELECT
      count(*) AS count
    FROM
      layers
      JOIN data ON data.id = layers.id
    WHERE
      layers.project_id = ?0
  )
UPDATE layers
SET
  name = data.name,
  description = data.description,
  updated_at = ?4
FROM
  data,
  held
WHERE
  layers.project_id = ?0
  AND layers.id = data.id
  AND held.count = (
    SELECT
      count(*)
    FROM
      data
  )
RETURNING
  layers.*;
