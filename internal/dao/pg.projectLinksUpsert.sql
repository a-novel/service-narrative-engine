-- The write applies only while the links still hold the updated_at the request names. A request that names none
-- inserts the first links, and conflicts with links already written, since updated_at = NULL never holds.
INSERT INTO
  project_links (project_id, links, updated_at)
VALUES
  (?0, ?1::jsonb, ?3)
ON CONFLICT (project_id) DO UPDATE
SET
  links = EXCLUDED.links,
  updated_at = EXCLUDED.updated_at
WHERE
  project_links.updated_at = ?2
RETURNING
  updated_at;
