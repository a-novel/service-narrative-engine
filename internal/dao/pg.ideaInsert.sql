INSERT INTO
  ideas (
    id,
    owner_id,
    seed,
    story_type,
    genre,
    title,
    created_at,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?6)
RETURNING
  *;
