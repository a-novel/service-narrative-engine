-- The recap applies while it read a version of the cell no older than the stored recap's, and no newer
-- than the cell's own. The parts of one WITH share a snapshot, so the outer read sees the cell either way:
-- a cell the project lacks returns no row, and a refused write returns false.
WITH
  updated AS (
    UPDATE cells
    SET
      recap = ?3::jsonb,
      recap_cell_updated_at = ?4
    WHERE
      project_id = ?0
      AND scene_id = ?1
      AND layer_id = ?2
      AND updated_at >= ?4
      AND (
        recap_cell_updated_at IS NULL
        OR recap_cell_updated_at <= ?4
      )
    RETURNING
      1
  )
SELECT
  EXISTS (
    SELECT
    FROM
      updated
  ) AS written
FROM
  cells
WHERE
  project_id = ?0
  AND scene_id = ?1
  AND layer_id = ?2;
