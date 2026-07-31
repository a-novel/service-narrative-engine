WITH
  inserted_idea AS (
    INSERT INTO
      ideas (id, owner_id, created_at)
    VALUES
      (?0, ?1, ?5)
    RETURNING
      id,
      owner_id,
      created_at
  ),
  inserted_version AS (
    INSERT INTO
      idea_versions (id, idea_id, seed, genre, title, created_at)
    SELECT
      inserted_idea.id,
      inserted_idea.id,
      ?2,
      ?3,
      ?4,
      ?5
    FROM
      inserted_idea
    RETURNING
      seed,
      genre,
      title
  )
SELECT
  inserted_idea.id,
  inserted_idea.owner_id,
  inserted_version.seed,
  inserted_version.genre,
  inserted_version.title,
  inserted_idea.created_at,
  NULL::timestamp with time zone AS updated_at
FROM
  inserted_idea
  CROSS JOIN inserted_version;
