-- A fixture runs after its matching migration and seeds data for the next migration. The fixture beat
-- holds the scene the previous fixture seeded, so it adds a second scene beside it.
INSERT INTO
  scenes (id, project_id, beat_id, position, title)
VALUES
  (
    '00000000-0000-0000-0000-000000000012',
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000011',
    1,
    'fixture second scene'
  );
