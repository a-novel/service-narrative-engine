-- The recap applies while it read a version of the beat no older than the stored recap's, and no newer
-- than the beat's own. The parts of one WITH share a snapshot, so the outer read sees the beat either way:
-- a beat the project lacks returns no row, and a refused write returns false.
WITH
  updated AS (
    UPDATE beats
    SET
      recap = ?2::jsonb,
      recap_beat_updated_at = ?3,
      recap_sources = ?4::jsonb
    WHERE
      project_id = ?0
      AND id = ?1
      AND updated_at >= ?3
      AND (
        recap_beat_updated_at IS NULL
        OR recap_beat_updated_at <= ?3
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
  beats
WHERE
  project_id = ?0
  AND id = ?1;
