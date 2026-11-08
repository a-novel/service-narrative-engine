-- The picture and its version change in one statement, so a read never finds one without the other.
WITH
  stored AS (
    INSERT INTO
      project_pictures (project_id, picture, data, media_type)
    VALUES
      (?0, ?1, ?2, ?3)
    ON CONFLICT (project_id, picture) DO UPDATE
    SET
      data = EXCLUDED.data,
      media_type = EXCLUDED.media_type
  )
UPDATE projects
SET
  logo_updated_at = CASE
    WHEN ?1 = 'logo' THEN ?4
    ELSE logo_updated_at
  END,
  background_updated_at = CASE
    WHEN ?1 = 'background' THEN ?4
    ELSE background_updated_at
  END
WHERE
  id = ?0
RETURNING
  *;
