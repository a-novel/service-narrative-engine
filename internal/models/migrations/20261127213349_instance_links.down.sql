-- The links of every instance are lost, and no later migration brings them back.
--
-- The bytes of a row stop counting its links first: the expression names the column, which could not go
-- otherwise.
ALTER TABLE instances
ALTER COLUMN bytes
SET
  EXPRESSION AS (
    octet_length(plugin) + octet_length(version) + octet_length(name) + octet_length(content::text)
  );

ALTER TABLE instances
DROP COLUMN IF EXISTS links_updated_at,
DROP COLUMN IF EXISTS links;

-- The old expression rewrites every row and fires no trigger, so each project is recounted without the links
-- of its instances.
UPDATE projects
SET
  stored_bytes = fresh.bytes
FROM
  (
    SELECT
      projects.id,
      coalesce(sum(held.bytes), 0)::bigint AS bytes
    FROM
      projects
      LEFT JOIN (
        SELECT
          project_id,
          bytes
        FROM
          instances
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          project_links
      ) AS held ON held.project_id = projects.id
    GROUP BY
      projects.id
  ) AS fresh
WHERE
  projects.id = fresh.id
  AND projects.stored_bytes <> fresh.bytes;
