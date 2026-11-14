SELECT
  project_id,
  data,
  media_type,
  encode(checksum, 'hex') AS checksum,
  updated_at
FROM
  project_logos
WHERE
  project_id = ?0;
