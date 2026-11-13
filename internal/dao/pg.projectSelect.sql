SELECT
  projects.*,
  logos.updated_at AS logo_updated_at,
  encode(logos.checksum, 'hex') AS logo_checksum
FROM
  projects
  LEFT JOIN project_logos AS logos ON logos.project_id = projects.id
WHERE
  projects.id = ?0;
