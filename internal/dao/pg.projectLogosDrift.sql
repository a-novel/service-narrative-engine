-- The full join pairs the logos of the owner's projects with the held ones. A logo held with another checksum, or not
-- held, drifted, and a held ID that pairs with no logo of the owner is removed. A removed row carries no checksum.
WITH
  held AS (
    SELECT
      key::uuid AS project_id,
      value AS checksum
    FROM
      jsonb_each_text(?1::jsonb)
  ),
  owned AS (
    SELECT
      logos.project_id,
      encode(logos.checksum, 'hex') AS checksum
    FROM
      project_logos AS logos
      JOIN projects ON projects.id = logos.project_id
    WHERE
      projects.owner_id = ?0
  )
SELECT
  coalesce(owned.project_id, held.project_id) AS project_id,
  owned.checksum
FROM
  owned
  FULL JOIN held ON held.project_id = owned.project_id
WHERE
  owned.project_id IS NULL
  OR held.project_id IS NULL
  OR held.checksum <> owned.checksum
LIMIT
  ?2;
