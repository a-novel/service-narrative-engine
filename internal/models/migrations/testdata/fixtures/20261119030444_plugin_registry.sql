-- A fixture runs after its matching migration and seeds data for the rollback. A version with its messages makes
-- the down migration drop both tables while the foreign key holds rows.
INSERT INTO
  plugin_versions (
    plugin,
    major,
    minor,
    patch,
    schema,
    exports,
    migration,
    ui,
    published_at,
    published_by
  )
VALUES
  (
    'fixture-plugin',
    1,
    0,
    0,
    '{"type": "object"}',
    '{}',
    'export default (content) => ({ migrated: content });',
    'export default { mount() {} };',
    '2026-11-19T00:00:00Z',
    'fixture'
  );

INSERT INTO
  plugin_messages (
    plugin,
    major,
    minor,
    patch,
    prerelease,
    language,
    title,
    description,
    messages
  )
VALUES
  (
    'fixture-plugin',
    1,
    0,
    0,
    '',
    'en',
    'Fixture plugin',
    'A plugin for the migration tests.',
    '{}'
  );
