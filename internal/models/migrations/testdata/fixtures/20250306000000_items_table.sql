-- A fixture runs after its matching migration and seeds data for the next migration.
INSERT INTO
  items (id, name, description)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'fixture item',
    'fixture data for the original schema'
  );
