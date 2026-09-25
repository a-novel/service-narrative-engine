SELECT
  id,
  project_id,
  seed,
  genre,
  title,
  created_at
FROM
  idea_versions
WHERE
  project_id = ?0
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  ?1;
