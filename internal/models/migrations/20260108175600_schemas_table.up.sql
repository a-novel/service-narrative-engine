-- Create enum type for schema source
CREATE TYPE schema_source AS ENUM('USER', 'AI', 'FORK', 'EXTERNAL');

CREATE TABLE schemas (
  -- The id of the current version of the schema.
  id uuid NOT NULL,
  -- Project id links multiple versions of the same schema together.
  project_id uuid NOT NULL,
  -- ID of the user who owns this schema version (nullable).
  owner uuid,
  -- The id of the module used to create the schema.
  module_id text NOT NULL,
  -- The namespace of the module used to create the schema.
  module_namespace text NOT NULL,
  -- The version of the module used to create the schema.
  module_version text NOT NULL,
  -- The preversion of the module used to create the schema.
  module_preversion text NOT NULL DEFAULT '',
  -- The source of this schema version.
  source schema_source NOT NULL,
  -- The content of the story. Null indicates the data for the given module has been cleared.
  data jsonb DEFAULT NULL,
  created_at timestamp(0) with time zone NOT NULL,
  PRIMARY KEY (id, project_id)
)
PARTITION BY
  HASH (project_id);

-- Create initial partitions (8 partitions for good distribution)
CREATE TABLE schemas_p0 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 0);

CREATE TABLE schemas_p1 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 1);

CREATE TABLE schemas_p2 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 2);

CREATE TABLE schemas_p3 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 3);

CREATE TABLE schemas_p4 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 4);

CREATE TABLE schemas_p5 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 5);

CREATE TABLE schemas_p6 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 6);

CREATE TABLE schemas_p7 PARTITION OF schemas FOR
VALUES
WITH
  (MODULUS 8, REMAINDER 7);

-- Composite index for history queries (get latest, list versions)
-- This index covers all WHERE clause columns and the ORDER BY, enabling index-only scans
CREATE INDEX idx_schemas_history ON schemas (
  project_id,
  module_id,
  module_namespace,
  created_at DESC
);

-- Index on project_id alone for partition pruning and project-wide operations
CREATE INDEX idx_schemas_project ON schemas (project_id);

-- Increase statistics for better query planning on frequently filtered columns
ALTER TABLE schemas
ALTER COLUMN project_id
SET
  STATISTICS 1000;

ALTER TABLE schemas
ALTER COLUMN module_id
SET
  STATISTICS 1000;

ALTER TABLE schemas
ALTER COLUMN module_namespace
SET
  STATISTICS 1000;
