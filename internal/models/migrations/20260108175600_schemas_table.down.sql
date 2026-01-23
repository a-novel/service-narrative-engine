-- Drop indexes first
DROP INDEX IF EXISTS idx_schemas_history;

DROP INDEX IF EXISTS idx_schemas_project;

-- Drop partitions (they will be dropped automatically with the parent table, but explicit is clearer)
DROP TABLE IF EXISTS schemas_p0;

DROP TABLE IF EXISTS schemas_p1;

DROP TABLE IF EXISTS schemas_p2;

DROP TABLE IF EXISTS schemas_p3;

DROP TABLE IF EXISTS schemas_p4;

DROP TABLE IF EXISTS schemas_p5;

DROP TABLE IF EXISTS schemas_p6;

DROP TABLE IF EXISTS schemas_p7;

-- Drop the main table
DROP TABLE IF EXISTS schemas;

-- Drop the enum type
DROP TYPE IF EXISTS schema_source;
