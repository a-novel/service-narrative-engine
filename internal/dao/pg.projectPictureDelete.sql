-- The picture and its version go in one statement, so a read never finds one without the other.
WITH
  deleted AS (
    DELETE FROM project_pictures
    WHERE
      project_id = ?0
      AND picture = ?1
  )
UPDATE projects
SET
  logo_updated_at = CASE
    WHEN ?1 = 'logo' THEN NULL
    ELSE logo_updated_at
  END,
  background_updated_at = CASE
    WHEN ?1 = 'background' THEN NULL
    ELSE background_updated_at
  END
WHERE
  id = ?0
RETURNING
  *;
