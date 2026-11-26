-- A fixture runs after its matching migration and seeds data for the rollback. The fixture project takes two
-- instances, a default among them, and its links, so the down migration drops the column, both tables and their
-- triggers while they hold rows. The second instance has no name, which the schema allows.
INSERT INTO
  instances (
    id,
    project_id,
    plugin,
    version,
    name,
    content,
    recovery,
    created_at,
    updated_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000031',
    '00000000-0000-0000-0000-000000000001',
    'scenes-composer',
    '1.0.0',
    'Fixture composer',
    '{"scenes": []}',
    NULL,
    '2026-11-26T00:00:00.000001Z',
    '2026-11-26T00:00:00.000001Z'
  ),
  (
    '00000000-0000-0000-0000-000000000032',
    '00000000-0000-0000-0000-000000000001',
    'scenes-composer',
    '1.1.0-dev.1',
    '',
    '{"scenes": [{"id": "a"}]}',
    '{"scenes": []}',
    '2026-11-26T00:00:00.000002Z',
    '2026-11-26T00:00:00.000003Z'
  );

UPDATE projects
SET
  default_instance_id = '00000000-0000-0000-0000-000000000031'
WHERE
  id = '00000000-0000-0000-0000-000000000001';

INSERT INTO
  project_links (project_id, links, updated_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    '[{"source": {"id": "00000000-0000-0000-0000-000000000031", "target": "scenes/a"}, "destination": {"id": "00000000-0000-0000-0000-000000000032", "target": "scenes/a"}, "relationship": "therefore"}]',
    '2026-11-26T00:00:00.000004Z'
  );

-- A second project holds a default instance and links too. The checks below delete it.
INSERT INTO
  projects (id, owner_id, title, language)
VALUES
  (
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-0000000000a1',
    'deleted fixture project',
    'en'
  );

INSERT INTO
  instances (
    id,
    project_id,
    plugin,
    version,
    name,
    content,
    created_at,
    updated_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000041',
    '00000000-0000-0000-0000-000000000002',
    'scenes-composer',
    '1.0.0',
    'Deleted fixture composer',
    '{}',
    '2026-11-26T00:00:00Z',
    '2026-11-26T00:00:00Z'
  );

UPDATE projects
SET
  default_instance_id = '00000000-0000-0000-0000-000000000041'
WHERE
  id = '00000000-0000-0000-0000-000000000002';

INSERT INTO
  project_links (project_id, links, updated_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000002',
    '[]',
    '2026-11-26T00:00:00Z'
  );

-- The checks fail the roundtrip unless the schema keeps what the service relies on.
DO $$
BEGIN
  -- The triggers count every instance and every links document in the stored bytes of their project.
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
    RAISE EXCEPTION 'a project counts stored bytes apart from its instances and its links';
  END IF;

  -- A project takes none but its own instances as its default.
  BEGIN
    UPDATE projects
    SET
      default_instance_id = '00000000-0000-0000-0000-000000000041'
    WHERE
      id = '00000000-0000-0000-0000-000000000001';

    RAISE EXCEPTION 'a project took the instance of another project as its default';
  EXCEPTION
    WHEN foreign_key_violation THEN
      NULL;
  END;

  -- The default instance of a project cannot be deleted. PostgreSQL raises restrict_violation for it, the
  -- code the instance delete statement maps to its refusal.
  BEGIN
    DELETE FROM instances
    WHERE
      id = '00000000-0000-0000-0000-000000000031';

    RAISE EXCEPTION 'the default instance of a project was deleted';
  EXCEPTION
    WHEN restrict_violation THEN
      NULL;
  END;

  -- Deleting a project crosses the cascade to its instances and the restriction on its default: it succeeds,
  -- and takes the instances and the links with it.
  DELETE FROM projects
  WHERE
    id = '00000000-0000-0000-0000-000000000002';

  IF EXISTS (
    SELECT
      1
    FROM
      instances
    WHERE
      project_id = '00000000-0000-0000-0000-000000000002'
  ) OR EXISTS (
    SELECT
      1
    FROM
      project_links
    WHERE
      project_id = '00000000-0000-0000-0000-000000000002'
  ) THEN
    RAISE EXCEPTION 'a deleted project left its instances or its links behind';
  END IF;
END
$$;
