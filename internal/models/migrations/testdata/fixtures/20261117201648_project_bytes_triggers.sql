-- A fixture runs after its matching migration. This one adds bytes the project does not hold, as a write the
-- previous release served during the roll would, so the next migration has a drifted count to correct.
UPDATE projects
SET
  stored_bytes = stored_bytes + 100
WHERE
  id = '00000000-0000-0000-0000-000000000001';
