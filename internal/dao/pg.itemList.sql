-- Item IDs are UUIDv7, which sort by creation time, so the primary key serves this order. A page
-- starts after the last ID of the previous one, and a deleted ID still marks its place.
SELECT
  *
FROM
  items
WHERE
  id < ?1
ORDER BY
  id DESC
LIMIT
  ?0;
