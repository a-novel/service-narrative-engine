SELECT
  count(*)::integer AS count
FROM
  projects
WHERE
  owner_id = ?0;
