INSERT INTO
  modules (
    id,
    namespace,
    version,
    preversion,
    description,
    schema,
    ui,
    created_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7)
RETURNING
  *;
