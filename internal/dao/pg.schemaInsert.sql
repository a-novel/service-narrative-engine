INSERT INTO
  schemas (
    id,
    project_id,
    owner,
    module_id,
    module_namespace,
    module_version,
    module_preversion,
    source,
    data,
    created_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9)
RETURNING
  *;
