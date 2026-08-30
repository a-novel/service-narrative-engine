SELECT DISTINCT
  ON (key) id,
  project_id,
  key,
  value,
  created_at
FROM
  step_values
WHERE
  project_id = ?0
ORDER BY
  key,
  created_at DESC,
  id DESC;
