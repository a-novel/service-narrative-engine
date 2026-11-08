-- RETURNING cannot join, so the outer query joins the picture versions to the updated row.
WITH
  updated AS (
    UPDATE projects
    SET
      title = ?1,
      premise = ?2,
      language = ?3,
      updated_at = ?4
    WHERE
      id = ?0
    RETURNING
      *
  )
SELECT
  updated.*,
  logo.updated_at AS logo_updated_at,
  background.updated_at AS background_updated_at
FROM
  updated
  LEFT JOIN project_pictures AS logo ON logo.project_id = updated.id
  AND logo.picture = 'logo'
  LEFT JOIN project_pictures AS background ON background.project_id = updated.id
  AND background.picture = 'background';
