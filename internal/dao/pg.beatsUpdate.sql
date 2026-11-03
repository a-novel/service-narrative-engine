-- The update runs only when the project holds every beat it names, so it updates all of them or none.
-- The project's stored bytes take the difference between the beats' new bytes and their old ones.
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
  ),
  updated AS (
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
      beats.*,
      new.bytes - old.bytes AS delta
  ),
  -- Grouping by project yields no row when the batch updates none, which leaves the count as it is.
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + change.delta
    FROM
      (
        SELECT
          project_id,
          sum(delta) AS delta
        FROM
          updated
        GROUP BY
          project_id
      ) AS change
    WHERE
      projects.id = change.project_id
  )
SELECT
  *
FROM
  updated;
