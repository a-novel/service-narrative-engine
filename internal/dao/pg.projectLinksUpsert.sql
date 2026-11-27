-- The write applies only while the links still hold the updated_at the request names. A project that never wrote
-- its links holds no row, so its first write names none: the insert runs when the request names none, or when a
-- row exists for the update to compare. A write that names another value changes nothing and returns no row.
INSERT INTO
  project_links (project_id, links, updated_at)
SELECT
  ?0,
  ?1::jsonb,
  ?3
WHERE
  ?2::timestamp with time zone IS NULL
  OR EXISTS (
    SELECT
      1
    FROM
      project_links
    WHERE
      project_id = ?0
  )
ON CONFLICT (project_id) DO UPDATE
SET
  links = EXCLUDED.links,
  updated_at = EXCLUDED.updated_at
WHERE
  project_links.updated_at = ?2
RETURNING
  updated_at;
