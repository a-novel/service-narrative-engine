-- A project that never wrote its links holds no row. It reads as an empty list no write has stamped.
SELECT
  projects.id AS project_id,
  coalesce(project_links.links, '[]') AS links,
  project_links.updated_at
FROM
  projects
  LEFT JOIN project_links ON project_links.project_id = projects.id
WHERE
  projects.id = ?0;
