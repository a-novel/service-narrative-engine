-- The caller gives each instance its ID, and its client tells the instances it sent apart by their place, so they
-- come back in the order of the request.
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
    RETURNING
      id,
      project_id,
      plugin,
      version,
      name,
      created_at,
      updated_at
  )
SELECT
  inserted.*
FROM
  inserted
  JOIN data ON data.id = inserted.id
ORDER BY
  data.position;
