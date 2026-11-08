-- The picture and its version go in one row. A statement reads the tables as they stood before it, so the
-- versions leave out the removed picture.
WITH
  deleted AS (
    DELETE FROM project_pictures
    WHERE
      project_id = ?0
      AND picture = ?1
  ),
  versions AS (
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
