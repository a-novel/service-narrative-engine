-- The write applies only while the instance still holds the updated_at the request names. The outer query reads
-- the instance as it stood before the write, so an instance the project holds returns a row whether the write
-- applied or not, and stale tells the two apart.
WITH
  written AS (
    UPDATE instances
    SET
      version = ?2,
      content = ?3::jsonb,
      updated_at = ?5
    WHERE
      project_id = ?0
      AND id = ?1
      AND updated_at = ?4
    RETURNING
      id,
      version,
      updated_at
  )
SELECT
  instances.id,
  instances.project_id,
  instances.plugin,
  instances.name,
  instances.created_at,
  written.version,
  written.updated_at,
  written.id IS NULL AS stale
FROM
  instances
  LEFT JOIN written ON written.id = instances.id
WHERE
  instances.project_id = ?0
  AND instances.id = ?1;
