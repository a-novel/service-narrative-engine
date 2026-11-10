-- A fixture runs after its matching migration and seeds data for the next migration. The labels give the rollback
-- populated columns to drop.
UPDATE projects
SET
  tags = '{noir}',
  genres = '{thriller}'
WHERE
  id = '00000000-0000-0000-0000-000000000001';

-- A logo and a background of one project break the next migration's key unless it deletes the background.
INSERT INTO
  project_pictures (project_id, picture, data, media_type, updated_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'logo',
    '\x00',
    'image/webp',
    CURRENT_TIMESTAMP
  ),
  (
    '00000000-0000-0000-0000-000000000001',
    'background',
    '\x00',
    'image/jpeg',
    CURRENT_TIMESTAMP
  );
