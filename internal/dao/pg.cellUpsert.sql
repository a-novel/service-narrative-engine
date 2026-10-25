-- Revision 0 inserts, any other revision updates: each branch matches only the state it expects.
WITH
  updated AS (
    UPDATE cells
    SET
      content = ?3::jsonb,
      revision = revision + 1,
      updated_at = ?4
    WHERE
      scene_id = ?0
      AND layer_id = ?1
      AND revision = ?2
    RETURNING
      *
  ),
  inserted AS (
    INSERT INTO
      cells (scene_id, layer_id, content, updated_at)
    SELECT
      ?0,
      ?1,
      ?3::jsonb,
      ?4
    WHERE
      ?2 = 0
    ON CONFLICT (scene_id, layer_id) DO NOTHING
    RETURNING
      *
  )
SELECT
  *
FROM
  updated
UNION ALL
SELECT
  *
FROM
  inserted;
