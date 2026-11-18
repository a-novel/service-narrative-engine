-- While the triggers' release rolled out, the previous release kept serving and its statements still added their
-- own bytes, so a write it served counted twice. This recount sets every project's stored bytes to the sum of the
-- bytes its beats, scenes, layers and cells hold, which is what the triggers keep from now on.
--
-- It updates projects alone, so it fires none of the triggers, and only the projects whose count drifted. The
-- table holds a few rows, so one statement in the migration is enough; a larger one would recount in batches.
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
          beats
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          scenes
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          layers
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          cells
      ) AS held ON held.project_id = projects.id
    GROUP BY
      projects.id
  ) AS fresh
WHERE
  projects.id = fresh.id
  AND projects.stored_bytes <> fresh.bytes;
