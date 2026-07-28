INSERT INTO
  step_values (
    id,
    idea_id,
    engine_version_id,
    step_key,
    value,
    created_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5)
RETURNING
  *;
