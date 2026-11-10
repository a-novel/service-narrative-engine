SELECT
  projects.*,
  logos.updated_at AS logo_updated_at
FROM
  projects
  LEFT JOIN project_logos AS logos ON logos.project_id = projects.id
WHERE
  projects.id = ?0;
