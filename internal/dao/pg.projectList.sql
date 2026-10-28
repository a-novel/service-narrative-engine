-- Project IDs are UUIDv7, which sort by creation time, so the owner index serves this order.
SELECT
  *,
  count(*) OVER () AS total
FROM
  projects
WHERE
  owner_id = ?0
ORDER BY
  id DESC
LIMIT
  ?1
OFFSET
  ?2;
