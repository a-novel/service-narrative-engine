-- A fixture runs after its matching migration and seeds data for the next migration.
INSERT INTO
  projects (id, owner_id, title, language)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-0000000000a1',
    'fixture project',
    'en'
  );

INSERT INTO
  scenes (id, project_id, position, title)
VALUES
  (
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000001',
    0,
    'fixture scene'
  );

INSERT INTO
  layers (id, project_id, position, kind, name)
VALUES
  (
    '00000000-0000-0000-0000-000000000021',
    '00000000-0000-0000-0000-000000000001',
    0,
    'manuscript',
    'Manuscript'
  );

INSERT INTO
  cells (scene_id, layer_id, content)
VALUES
  (
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000021',
    '{"blocks": []}'
  );
