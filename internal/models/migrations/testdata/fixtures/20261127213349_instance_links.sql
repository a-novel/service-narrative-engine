-- A fixture runs after its matching migration and seeds data for the rollback. The first fixture instance writes
-- its links and the second never does, so the down migration drops both columns while one holds a list and a
-- stamp, and recounts a project whose bytes include them.
UPDATE instances
SET
  links = '[{"source": "scenes/a", "destination": {"id": "00000000-0000-0000-0000-000000000032", "target": "scenes/a"}, "relationship": "therefore"}]',
  links_updated_at = '2026-11-27T00:00:00.000001Z'
WHERE
  id = '00000000-0000-0000-0000-000000000031';

-- The checks fail the roundtrip unless the schema keeps what the service relies on.
DO $$
BEGIN
  -- An instance that never wrote its links holds an empty list and no stamp.
  IF NOT EXISTS (
    SELECT
      1
    FROM
      instances
    WHERE
      id = '00000000-0000-0000-0000-000000000032'
      AND links = '[]'
      AND links_updated_at IS NULL
  ) THEN
    RAISE EXCEPTION 'an instance that never wrote its links holds something else than an empty list and no stamp';
  END IF;

  -- The migration recounted every project, and the triggers count the links an instance writes after it.
  IF EXISTS (
    SELECT
      1
    FROM
      projects
      LEFT JOIN (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          (
            SELECT project_id, bytes FROM instances
            UNION ALL
            SELECT project_id, bytes FROM project_links
          ) AS held
        GROUP BY
          project_id
      ) AS fresh ON fresh.project_id = projects.id
    WHERE
      projects.stored_bytes <> coalesce(fresh.bytes, 0)
  ) THEN
    RAISE EXCEPTION 'a project counts stored bytes apart from its instances, their links and its links';
  END IF;

  -- The links of an instance count in its bytes.
  IF EXISTS (
    SELECT
      1
    FROM
      instances
    WHERE
      bytes <> octet_length(plugin) + octet_length(version) + octet_length(name) + octet_length(content::text) + octet_length(links::text)
  ) THEN
    RAISE EXCEPTION 'the bytes of an instance leave its links out';
  END IF;
END
$$;
