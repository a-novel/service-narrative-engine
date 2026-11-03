-- Each new scene goes before the scene of its beat at its position, or last past the end, and new scenes
-- at one position keep the order of the request. A beat's scenes move down by the number of new scenes
-- placed at or before them. The positions unique constraint is deferred, so the shift and the insert run
-- in one statement. A beat the project does not hold breaks the beat foreign key, which fails the batch.
-- The project's stored bytes take the new scenes' bytes.
WITH
  data AS (
    SELECT
      *
    FROM
      unnest(
        ARRAY[?1]::uuid[],
        ARRAY[?2]::uuid[],
        ARRAY[?3]::integer[],
        ARRAY[?4]::text[],
        ARRAY[?5]::text[],
        ARRAY[?6]::text[]
      ) WITH ORDINALITY AS data (
        id,
        beat_id,
        position,
        title,
        description,
        prompt,
        ordinality
      )
  ),
  held AS (
    SELECT
      targets.beat_id,
      count(scenes.id)::integer AS count
    FROM
      (
        SELECT DISTINCT
          beat_id
        FROM
          data
      ) AS targets
      LEFT JOIN scenes ON scenes.project_id = ?0
      AND scenes.beat_id = targets.beat_id
    GROUP BY
      targets.beat_id
  ),
  placed AS (
    SELECT
      data.*,
      LEAST(data.position, held.count) AS slot
    FROM
      data
      JOIN held ON held.beat_id = data.beat_id
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position + (
        SELECT
          count(*)
        FROM
          placed
        WHERE
          placed.beat_id = scenes.beat_id
          AND placed.slot <= scenes.position
      )::integer
    WHERE
      scenes.project_id = ?0
      AND EXISTS (
        SELECT
          1
        FROM
          placed
        WHERE
          placed.beat_id = scenes.beat_id
          AND placed.slot <= scenes.position
      )
  ),
  inserted AS (
    INSERT INTO
      scenes (
        id,
        project_id,
        beat_id,
        position,
        title,
        description,
        prompt,
        updated_at
      )
    SELECT
      placed.id,
      ?0,
      placed.beat_id,
      (
        placed.slot + row_number() OVER (
          PARTITION BY
            placed.beat_id
          ORDER BY
            placed.slot,
            placed.ordinality
        ) - 1
      )::integer,
      placed.title,
      placed.description,
      placed.prompt,
      ?7
    FROM
      placed
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
