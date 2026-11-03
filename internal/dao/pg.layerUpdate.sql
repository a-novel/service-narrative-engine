-- The project's stored bytes take the difference between the layer's new bytes and its old ones.
WITH
  updated AS (
    UPDATE layers
    SET
      name = ?2,
      description = ?3,
      updated_at = ?4
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
