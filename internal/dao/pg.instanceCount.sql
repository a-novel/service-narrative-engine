SELECT
  count(*)::integer AS count
FROM
  instances
WHERE
  project_id = ?0;
