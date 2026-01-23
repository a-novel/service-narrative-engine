INSERT INTO
  projects (
    id,
    owner,
    lang,
    title,
    workflow,
    created_at,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4::text[], ?5, ?6)
RETURNING
  *;
