-- The update runs only when the project holds every layer it names, so it updates all of them or none.
-- The project's stored bytes take the difference between the layers' new bytes and their old ones.
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
  ),
  updated AS (
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
      layers.*,
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
