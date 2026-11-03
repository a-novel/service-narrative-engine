-- The project's stored bytes take the difference between the beat's new bytes and its old ones.
WITH
  updated AS (
    UPDATE beats
    SET
      title = ?2,
      color = ?3,
      description = ?4,
      prompt = ?5,
      updated_at = ?6
    WHERE
      project_id = ?0
      AND id = ?1
    RETURNING
      *,
      new.bytes - old.bytes AS delta
  ),
  counted AS (
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + updated.delta
    FROM
      updated
    WHERE
      projects.id = updated.project_id
  )
SELECT
  *
FROM
  updated;
