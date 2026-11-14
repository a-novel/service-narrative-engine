SELECT
  projects.*,
  encode(logos.checksum, 'hex') AS logo_checksum
FROM
  projects
  LEFT JOIN project_logos AS logos ON logos.project_id = projects.id
WHERE
  projects.id = ?0;
