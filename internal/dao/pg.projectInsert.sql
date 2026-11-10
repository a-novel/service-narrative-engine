-- A new project holds no logo, so it returns no logo version.
INSERT INTO
  projects (
    id,
    owner_id,
    title,
    premise,
    language,
    tags,
    genres,
    created_at,
    updated_at,
    edited_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?7, ?7)
RETURNING
  *;
