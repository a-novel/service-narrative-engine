-- The update runs only when the project holds every scene it names, so it updates all of them or none.
-- The project's stored bytes take the difference between the scenes' new bytes and their old ones.
WITH
  data AS (
    SELECT
      *
    FROM
      unnest(
        ARRAY[?1]::uuid[],
        ARRAY[?2]::text[],
        ARRAY[?3]::text[],
        ARRAY[?4]::text[]
      ) AS data (id, title, description, prompt)
  ),
  held AS (
    SELECT
      count(*) AS count
    FROM
      scenes
      JOIN data ON data.id = scenes.id
    WHERE
      scenes.project_id = ?0
  ),
  updated AS (
    UPDATE scenes
    SET
      title = data.title,
      description = data.description,
      prompt = data.prompt,
      updated_at = ?5
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
      scenes.*,
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
