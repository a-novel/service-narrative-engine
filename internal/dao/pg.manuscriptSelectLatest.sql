SELECT
  *
FROM
  manuscripts
WHERE
  idea_id = ?0
ORDER BY
  COALESCE(updated_at, created_at) DESC,
  id DESC
LIMIT
  1;
