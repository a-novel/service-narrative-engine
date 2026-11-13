-- The full join pairs the owner's projects with the held ones. A project held at another instant, or not held, drifted,
-- and a held ID that pairs with no project of the owner is removed. Each row is one of the two, so the removed ID is
-- set only on a row without a project. Instants compare to the millisecond, the precision a JavaScript date keeps, so
-- a client holds updatedAt in sync whether it sends it as served or as a date; two changes within one millisecond
-- read as one until the next.
WITH
  held AS (
    SELECT
      key::uuid AS id,
      value::timestamptz AS updated_at
    FROM
      jsonb_each_text(?1::jsonb)
  ),
  owned AS (
    SELECT
      *
    FROM
      projects
    WHERE
      owner_id = ?0
  )
SELECT
  owned.*,
  CASE
    WHEN owned.id IS NULL THEN held.id
  END AS removed_id
FROM
  owned
  FULL JOIN held ON held.id = owned.id
WHERE
  owned.id IS NULL
  OR held.id IS NULL
  OR date_trunc('milliseconds', held.updated_at) <> date_trunc('milliseconds', owned.updated_at)
LIMIT
  ?2;
