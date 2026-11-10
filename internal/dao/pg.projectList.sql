-- The owner index serves this order. Project IDs are UUIDv7, which sort by creation time, so two projects
-- edited at the same instant list newest first.
SELECT
  projects.*,
  logos.updated_at AS logo_updated_at,
  count(*) OVER () AS total
FROM
  projects
  LEFT JOIN project_logos AS logos ON logos.project_id = projects.id
WHERE
  projects.owner_id = ?0
ORDER BY
  projects.edited_at DESC,
  projects.id DESC
LIMIT
  ?1
OFFSET
  ?2;
