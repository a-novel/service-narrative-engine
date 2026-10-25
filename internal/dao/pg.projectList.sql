-- Project IDs are UUIDv7, which sort by creation time, so the owner index serves this order. A page
-- starts after the last ID of the previous one, and a deleted ID still marks its place.
SELECT
  *
FROM
  projects
WHERE
  owner_id = ?0
  AND id < ?2
ORDER BY
  id DESC
LIMIT
  ?1;
