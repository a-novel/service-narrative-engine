SELECT
  count(*) AS count
FROM
  cells
WHERE
  project_id = ?0;
