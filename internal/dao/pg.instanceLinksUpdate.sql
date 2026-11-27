-- The write applies only while the instance still holds the links_updated_at the request names: none for an
-- instance that never wrote its links, which IS NOT DISTINCT FROM matches where an equality never would. It
-- leaves updated_at, the stamp a content write names, as it was. The outer query reads the instance as it stood
-- before the write, so an instance the project holds returns a row whether the write applied or not, and stale
-- tells the two apart.
WITH
  written AS (
    UPDATE instances
    SET
      links = ?2::jsonb,
      links_updated_at = ?4
    WHERE
      project_id = ?0
      AND id = ?1
      AND links_updated_at IS NOT DISTINCT FROM ?3
    RETURNING
      id,
      links_updated_at
  )
SELECT
  written.links_updated_at,
  written.id IS NULL AS stale
FROM
  instances
  LEFT JOIN written ON written.id = instances.id
WHERE
  instances.project_id = ?0
  AND instances.id = ?1;
