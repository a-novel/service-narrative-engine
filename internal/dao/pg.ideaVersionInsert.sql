INSERT INTO
  idea_versions (id, idea_id, seed, genre, title, created_at)
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5)
RETURNING
  *;
