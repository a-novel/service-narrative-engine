-- An ID starts with its creation time, so the key lists the instances of a project oldest first.
SELECT
  id,
  project_id,
  plugin,
  version,
  name,
  created_at,
  updated_at
FROM
  instances
WHERE
  project_id = ?0
ORDER BY
  id
LIMIT
  ?1;
