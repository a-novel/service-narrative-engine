SELECT
  projects.*,
  logo.updated_at AS logo_updated_at,
  background.updated_at AS background_updated_at
FROM
  projects
  LEFT JOIN project_pictures AS logo ON logo.project_id = projects.id
  AND logo.picture = 'logo'
  LEFT JOIN project_pictures AS background ON background.project_id = projects.id
  AND background.picture = 'background'
WHERE
  projects.id = ?0;
