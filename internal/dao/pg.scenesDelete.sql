-- Each scene after a deleted one in its beat moves up by the number of deleted scenes before it in that
-- beat, so positions stay dense. The update still sees the rows the delete removes, so it skips them by ID.
WITH
  deleted AS (
    DELETE FROM scenes
    WHERE
      project_id = ?0
      AND id = ANY (ARRAY[?1]::uuid[])
    RETURNING
      *
  ),
  shifted AS (
    UPDATE scenes
    SET
      position = scenes.position - (
        SELECT
          count(*)
        FROM
          deleted
        WHERE
          deleted.beat_id = scenes.beat_id
          AND deleted.position < scenes.position
      )::integer
    WHERE
      scenes.project_id = ?0
      AND scenes.id <> ALL (ARRAY[?1]::uuid[])
      AND EXISTS (
        SELECT
          1
        FROM
          deleted
        WHERE
          deleted.beat_id = scenes.beat_id
          AND deleted.position < scenes.position
      )
  )
SELECT
  *
FROM
  deleted;
