-- The batch joins the project only when the project then holds as many instances as the request allows at most,
-- and none of it does otherwise. The first instance of the request becomes the default of a project that has
-- none. That project holds no instance: a default is never unset, and its instance is never deleted.
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
      ) WITH ORDINALITY AS data (id, plugin, version, name, content, position)
  ),
  inserted AS (
    INSERT INTO
      instances (
        id,
        project_id,
        plugin,
        version,
        name,
        content,
        created_at,
        updated_at
      )
    SELECT
      data.id,
      ?0,
      data.plugin,
      data.version,
      data.name,
      data.content::jsonb,
      ?6,
      ?6
    FROM
      data
    WHERE
      (
        SELECT
          count(*)
        FROM
          instances
        WHERE
          project_id = ?0
      ) + (
        SELECT
          count(*)
        FROM
          data
      ) <= ?7
    RETURNING
      id,
      project_id,
      plugin,
      version,
      name,
      created_at,
      updated_at
  ),
  defaulted AS (
    UPDATE projects
    SET
      default_instance_id = data.id
    FROM
      inserted
      JOIN data ON data.id = inserted.id
    WHERE
      projects.id = ?0
      AND projects.default_instance_id IS NULL
      AND data.position = 1
  )
SELECT
  inserted.*
FROM
  inserted
  JOIN data ON data.id = inserted.id
ORDER BY
  data.position;
