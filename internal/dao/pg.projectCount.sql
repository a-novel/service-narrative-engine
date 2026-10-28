SELECT
  count(*) AS count
FROM
  projects
WHERE
  owner_id = ?0;
