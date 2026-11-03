-- A position past the last beat appends. The beats at or after the insert position move down to make
-- room. The positions unique constraint is deferred, so the shift and the insert run in one statement.
-- The project's stored bytes take the new beats' bytes.
WITH
  start AS (
    SELECT
      LEAST(?1, count(*))::integer AS position
    FROM
      beats
    WHERE
      project_id = ?0
  ),
  shifted AS (
    UPDATE beats
    SET
      position = beats.position + cardinality(ARRAY[?2]::uuid[])
    FROM
      start
    WHERE
      beats.project_id = ?0
      AND beats.position >= start.position
  ),
  inserted AS (
    INSERT INTO
      beats (
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
      start.position + (data.position - 1)::integer,
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
      ) WITH ORDINALITY AS data (id, title, color, description, prompt, position),
      start
    RETURNING
      *
  ),
  -- Grouping by project yields no row when the batch inserts none, which leaves the count as it is.
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + change.bytes
    FROM
      (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          inserted
        GROUP BY
          project_id
      ) AS change
    WHERE
      projects.id = change.project_id
  )
SELECT
  *
FROM
  inserted;
