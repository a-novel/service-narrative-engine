INSERT INTO
  items (id, name, description, created_at, updated_at)
VALUES
  (?0, ?1, ?2, ?3, ?3)
RETURNING
  *;
