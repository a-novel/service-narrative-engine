WITH
  inserted_project AS (
    INSERT INTO
      projects (id, owner_id, created_at)
    VALUES
      (?0, ?2, ?6)
    RETURNING
      id,
      owner_id,
      created_at
  ),
  inserted_version AS (
    INSERT INTO
      idea_versions (id, project_id, seed, genre, title, created_at)
    SELECT
      ?1,
      inserted_project.id,
      ?3,
      ?4,
      ?5,
      ?6
    FROM
      inserted_project
    RETURNING
      id,
      project_id,
      seed,
      genre,
      title,
      created_at
  )
SELECT
  inserted_project.id AS project_id,
  inserted_version.id AS version_id,
  inserted_project.owner_id,
  inserted_version.seed,
  inserted_version.genre,
  inserted_version.title,
  inserted_project.created_at AS project_created_at,
  inserted_version.created_at
FROM
  inserted_project
  CROSS JOIN inserted_version;
