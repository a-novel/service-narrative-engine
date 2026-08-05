INSERT INTO
  step_values (id, project_id, key, value, created_at)
VALUES
  (?0, ?1, ?2, ?3, ?4)
RETURNING
  *;
