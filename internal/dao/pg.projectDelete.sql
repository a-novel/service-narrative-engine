-- The outer query reads the pictures as they stood before the delete, so it returns the versions the project
-- held.
WITH
  deleted AS (
    DELETE FROM projects
    WHERE
      id = ?0
    RETURNING
      *
  )
SELECT
  deleted.*,
  logo.updated_at AS logo_updated_at,
  background.updated_at AS background_updated_at
FROM
  deleted
  LEFT JOIN project_pictures AS logo ON logo.project_id = deleted.id
  AND logo.picture = 'logo'
  LEFT JOIN project_pictures AS background ON background.project_id = deleted.id
  AND background.picture = 'background';
