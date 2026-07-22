-- id breaks ties on created_at. LIMIT/OFFSET over a partial order lets Postgres return a tied group
-- in a different sequence per query, so paging repeats some rows and skips others.
SELECT
  *
FROM
  items
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  ?0
OFFSET
  ?1;
