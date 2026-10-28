-- The layers at or after the insert position move down to make room. The positions unique constraint
-- is deferred, so the shift and the insert run in one statement.
WITH
  shifted AS (
    UPDATE layers
    SET
      position = position + 1
    WHERE
      project_id = ?1
      AND position >= ?2
  )
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
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6)
RETURNING
  *;
