-- The picture and its version change in one row. A statement reads the tables as they stood before it, so the
-- versions take the written picture from the insert and the other picture from its row.
WITH
  stored AS (
    INSERT INTO
      project_pictures (project_id, picture, data, media_type, updated_at)
    VALUES
      (?0, ?1, ?2, ?3, ?4)
    ON CONFLICT (project_id, picture) DO UPDATE
    SET
      data = EXCLUDED.data,
      media_type = EXCLUDED.media_type,
      updated_at = EXCLUDED.updated_at
    RETURNING
      picture,
      updated_at
  ),
  versions AS (
    SELECT
      picture,
      updated_at
    FROM
      stored
    UNION ALL
    SELECT
      picture,
      updated_at
    FROM
      project_pictures
    WHERE
      project_id = ?0
      AND picture <> ?1
  )
SELECT
  projects.*,
  logo.updated_at AS logo_updated_at,
  background.updated_at AS background_updated_at
FROM
  projects
  LEFT JOIN versions AS logo ON logo.picture = 'logo'
  LEFT JOIN versions AS background ON background.picture = 'background'
WHERE
  projects.id = ?0;
