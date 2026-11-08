SELECT
  project_pictures.*,
  CASE project_pictures.picture
    WHEN 'logo' THEN projects.logo_updated_at
    WHEN 'background' THEN projects.background_updated_at
  END AS updated_at
FROM
  project_pictures
  JOIN projects ON projects.id = project_pictures.project_id
WHERE
  project_pictures.project_id = ?0
  AND project_pictures.picture = ?1;
