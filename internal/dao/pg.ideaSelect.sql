SELECT
  project.id AS project_id,
  idea_version.id AS version_id,
  project.owner_id,
  idea_version.seed,
  idea_version.genre,
  idea_version.title,
  project.created_at AS project_created_at,
  idea_version.created_at
FROM
  projects AS project
  INNER JOIN LATERAL (
    SELECT
      *
    FROM
      idea_versions
    WHERE
      project_id = project.id
    ORDER BY
      created_at DESC,
      id DESC
    LIMIT
      1
  ) AS idea_version ON TRUE
WHERE
  project.id = ?0
  AND project.owner_id = ?1;
