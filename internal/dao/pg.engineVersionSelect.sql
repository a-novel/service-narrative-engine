SELECT
  engine_version.id,
  engine_version.engine_id,
  engine_version.version,
  engine_version.definition,
  engine_version.created_at,
  engine.kind AS engine_kind,
  engine.slug AS engine_slug
FROM
  engine_versions AS engine_version
  INNER JOIN engines AS engine ON engine.id = engine_version.engine_id
WHERE
  engine_version.id = ?0;
