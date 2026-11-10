-- A fixture runs after its matching migration and seeds data for the rollback. A PNG logo breaks the restored media
-- type check unless the down migration deletes it.
UPDATE project_logos
SET
  media_type = 'image/png'
WHERE
  project_id = '00000000-0000-0000-0000-000000000001';
