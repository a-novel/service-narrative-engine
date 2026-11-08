-- The owner index serves this order. Project IDs are UUIDv7, which sort by creation time, so two projects
-- edited at the same instant list newest first.
SELECT
  *,
  count(*) OVER () AS total
FROM
  projects
WHERE
  owner_id = ?0
ORDER BY
  edited_at DESC,
  id DESC
LIMIT
  ?1
OFFSET
  ?2;
