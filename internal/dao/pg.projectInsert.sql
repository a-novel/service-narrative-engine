INSERT INTO
  projects (
    id,
    owner_id,
    title,
    premise,
    language,
    created_at,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?5)
RETURNING
  *;
