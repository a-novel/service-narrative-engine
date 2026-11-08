-- The owner index serves this order. Project IDs are UUIDv7, which sort by creation time, so two projects
-- edited at the same instant list newest first.
SELECT
  projects.*,
  logo.updated_at AS logo_updated_at,
  background.updated_at AS background_updated_at,
  count(*) OVER () AS total
FROM
  projects
  LEFT JOIN project_pictures AS logo ON logo.project_id = projects.id
  AND logo.picture = 'logo'
  LEFT JOIN project_pictures AS background ON background.project_id = projects.id
  AND background.picture = 'background'
WHERE
  projects.owner_id = ?0
ORDER BY
  projects.edited_at DESC,
  projects.id DESC
LIMIT
  ?1
OFFSET
  ?2;
