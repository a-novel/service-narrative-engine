INSERT INTO
  ideas (
    id,
    owner_id,
    seed,
    genre,
    title,
    created_at,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?5)
RETURNING
  *;
