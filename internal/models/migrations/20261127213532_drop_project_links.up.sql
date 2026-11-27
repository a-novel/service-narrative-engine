-- Each instance stores its own links, and nothing reads the links document of a project any more. No link is
-- carried over: no release ever served the table.
DROP TABLE project_links;

-- Dropping a table fires no trigger, so each project still counts the bytes of the links it held. The recount
-- updates projects alone, and only the ones whose count changed.
UPDATE projects
SET
  stored_bytes = fresh.bytes
FROM
  (
    SELECT
      projects.id,
      coalesce(sum(instances.bytes), 0)::bigint AS bytes
    FROM
      projects
      LEFT JOIN instances ON instances.project_id = projects.id
    GROUP BY
      projects.id
  ) AS fresh
WHERE
  projects.id = fresh.id
  AND projects.stored_bytes <> fresh.bytes;
