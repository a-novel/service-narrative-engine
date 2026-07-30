SELECT
  *
FROM
  manuscripts
WHERE
  idea_id = ?0
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  1;
