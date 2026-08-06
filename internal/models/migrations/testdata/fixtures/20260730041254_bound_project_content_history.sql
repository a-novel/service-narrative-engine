UPDATE idea_versions
SET
  seed = '',
  genre = ''
WHERE
  project_id = '00000000-0000-0000-0000-000000000201';

UPDATE engine_versions
SET
  definition = '"opaque engine definition"'::jsonb
WHERE
  id = '00000000-0000-0000-0000-000000000100';

UPDATE step_values
SET
  value = '"opaque step value"'::jsonb
WHERE
  id = '00000000-0000-0000-0000-000000000203';
