-- An instance stores the static links whose source is one of its items: it is their importer. The links get a
-- column of their own, apart from the content, and a stamp of their own, so a links write and a content write
-- never make each other stale.
ALTER TABLE instances
/* The static links the instance is the importer of: a JSON array of {source, destination, relationship}. */
ADD COLUMN links jsonb NOT NULL DEFAULT '[]',
-- A links write names the value it read and applies only while the row still holds it, as a content write does
-- with updated_at. The column keeps PostgreSQL's microseconds for the same reason.
/* When the links last changed. Null for an instance that never wrote its links. */
ADD COLUMN links_updated_at timestamp with time zone;

-- The bytes of a row count its links, so the stored bytes triggers see every links write, and the links count in
-- the budget of their project.
ALTER TABLE instances
ALTER COLUMN bytes
SET
  EXPRESSION AS (
    octet_length(plugin) + octet_length(version) + octet_length(name) + octet_length(content::text) + octet_length(links::text)
  );

-- The new expression rewrites every row and fires no trigger, so each project is recounted: its instances now
-- count the two bytes of an empty list each. The statement updates projects alone, and only the ones whose
-- count changed.
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
