DROP TABLE items;

CREATE TYPE engine_kind AS ENUM('project', 'collection');

CREATE TABLE ideas (
  id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  seed text NOT NULL CHECK (seed <> ''),
  genre text NOT NULL CHECK (genre <> ''),
  title text CHECK (
    title IS NULL
    OR title <> ''
  ),
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone
);

CREATE INDEX ideas_owner_id_created_at_idx ON ideas (owner_id, created_at DESC);

CREATE TABLE engines (
  id uuid PRIMARY KEY NOT NULL,
  kind engine_kind NOT NULL,
  slug text NOT NULL CHECK (slug <> ''),
  UNIQUE (slug)
);

CREATE TABLE engine_versions (
  id uuid PRIMARY KEY NOT NULL,
  engine_id uuid NOT NULL,
  version text NOT NULL CHECK (version <> ''),
  definition jsonb NOT NULL CHECK (jsonb_typeof(definition) = 'object'),
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT engine_versions_engine_fk FOREIGN KEY (engine_id) REFERENCES engines (id),
  UNIQUE (engine_id, version)
);

-- Owner-scoped usage ledger for pricing provider work; service-jobs owns execution and results.
CREATE TABLE generation_calls (
  job_id uuid NOT NULL,
  attempt integer NOT NULL CHECK (attempt >= 1),
  owner_id uuid NOT NULL,
  provider text NOT NULL CHECK (provider <> ''),
  model text NOT NULL CHECK (model <> ''),
  input_tokens bigint NOT NULL CHECK (input_tokens >= 0),
  output_tokens bigint NOT NULL CHECK (output_tokens >= 0),
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT generation_calls_pkey PRIMARY KEY (job_id, attempt)
);

CREATE INDEX generation_calls_owner_id_created_at_idx ON generation_calls (owner_id, created_at DESC);

-- Step values are client-saved project content, independent of how each value was produced.
CREATE TABLE step_values (
  id uuid PRIMARY KEY NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL,
  step_key text NOT NULL CHECK (step_key <> ''),
  value jsonb NOT NULL CHECK (jsonb_typeof(value) = 'object'),
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT step_values_idea_fk FOREIGN KEY (idea_id) REFERENCES ideas (id),
  CONSTRAINT step_values_engine_version_fk FOREIGN KEY (engine_version_id) REFERENCES engine_versions (id),
  UNIQUE (idea_id, engine_version_id, step_key)
);

-- Manuscript metadata lives inside the opaque value; it is not Project Engine metadata.
CREATE TABLE manuscripts (
  id uuid PRIMARY KEY NOT NULL,
  idea_id uuid NOT NULL,
  value jsonb NOT NULL CHECK (jsonb_typeof(value) = 'object'),
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  CONSTRAINT manuscripts_idea_fk FOREIGN KEY (idea_id) REFERENCES ideas (id)
);

CREATE INDEX manuscripts_idea_id_idx ON manuscripts (idea_id);
