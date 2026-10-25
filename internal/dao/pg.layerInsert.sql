INSERT INTO
  layers (
    id,
    project_id,
    position,
    kind,
    name,
    description,
    updated_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6)
RETURNING
  *;
