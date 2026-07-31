SELECT
  idea.id,
  idea.owner_id,
  idea_version.seed,
  idea_version.genre,
  idea_version.title,
  idea.created_at,
  CASE
    WHEN idea_version.id = idea.id
    AND idea_version.created_at = idea.created_at THEN NULL
    ELSE idea_version.created_at
  END AS updated_at
FROM
  ideas AS idea
  INNER JOIN LATERAL (
    SELECT
      *
    FROM
      idea_versions
    WHERE
      idea_id = idea.id
    ORDER BY
      created_at DESC,
      id DESC
    LIMIT
      1
  ) AS idea_version ON TRUE
WHERE
  idea.id = ?0
  AND idea.owner_id = ?1;
