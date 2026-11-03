-- A position past the last layer appends. The layers at or after the insert position move down to make
-- room. The positions unique constraint is deferred, so the shift and the insert run in one statement.
-- The project's stored bytes take the new layer's bytes.
WITH
  start AS (
    SELECT
      LEAST(?2, count(*))::integer AS position
    FROM
      layers
    WHERE
      project_id = ?1
  ),
  shifted AS (
    UPDATE layers
    SET
      position = layers.position + 1
    FROM
      start
    WHERE
      layers.project_id = ?1
      AND layers.position >= start.position
  ),
  inserted AS (
    INSERT INTO
      layers (
        id,
        project_id,
        position,
        kind,
        name,
        description,
        updated_at
      )
    SELECT
      ?0,
      ?1,
      start.position,
      ?3,
      ?4,
      ?5,
      ?6
    FROM
      start
    RETURNING
      *
  ),
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + inserted.bytes
    FROM
      inserted
    WHERE
      projects.id = inserted.project_id
  )
SELECT
  *
FROM
  inserted;
