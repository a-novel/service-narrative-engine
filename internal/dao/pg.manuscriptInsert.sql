INSERT INTO
  manuscripts (id, project_id, value, created_at)
VALUES
  (?0, ?1, ?2, ?3)
RETURNING
  *;
