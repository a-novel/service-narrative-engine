DROP TABLE items;

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
  updated_at timestamp with time zone NOT NULL,
  UNIQUE (id, owner_id)
);

CREATE INDEX ideas_owner_id_created_at_idx ON ideas (owner_id, created_at DESC);

CREATE TABLE engine_versions (
  id uuid PRIMARY KEY NOT NULL,
  kind text NOT NULL CHECK (kind IN ('project', 'collection')),
  slug text NOT NULL CHECK (slug <> ''),
  version text NOT NULL CHECK (version <> ''),
  definition jsonb NOT NULL CHECK (jsonb_typeof(definition) = 'object'),
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT engine_versions_definition_kind_check CHECK (
    definition ? 'kind'
    AND definition ->> 'kind' = kind
  ),
  UNIQUE (slug, version)
);

CREATE TABLE generation_calls (
  id uuid PRIMARY KEY NOT NULL,
  job_id uuid NOT NULL,
  attempt integer NOT NULL CHECK (attempt >= 1),
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL,
  provider text NOT NULL CHECK (provider <> ''),
  provider_call_id text CHECK (
    provider_call_id IS NULL
    OR provider_call_id <> ''
  ),
  request_hash text NOT NULL CHECK (request_hash ~ '^[0-9a-f]{64}$'),
  model text NOT NULL CHECK (model <> ''),
  outcome text NOT NULL CHECK (
    outcome IN ('ok', 'refusal', 'incomplete', 'error')
  ),
  raw_output text,
  input_tokens bigint CHECK (
    input_tokens IS NULL
    OR input_tokens >= 0
  ),
  output_tokens bigint CHECK (
    output_tokens IS NULL
    OR output_tokens >= 0
  ),
  total_tokens bigint CHECK (
    total_tokens IS NULL
    OR total_tokens >= 0
  ),
  latency_ms bigint NOT NULL CHECK (latency_ms >= 0),
  refusal text CHECK (
    refusal IS NULL
    OR refusal <> ''
  ),
  error text CHECK (
    error IS NULL
    OR error <> ''
  ),
  created_at timestamp with time zone NOT NULL,
  completed_at timestamp with time zone NOT NULL,
  CONSTRAINT generation_calls_outcome_details_check CHECK (
    (
      outcome = 'ok'
      AND raw_output IS NOT NULL
      AND refusal IS NULL
      AND error IS NULL
    )
    OR (
      outcome = 'refusal'
      AND refusal IS NOT NULL
      AND error IS NULL
    )
    OR (
      outcome = 'incomplete'
      AND refusal IS NULL
      AND error IS NULL
    )
    OR (
      outcome = 'error'
      AND refusal IS NULL
      AND error IS NOT NULL
    )
  ),
  CONSTRAINT generation_calls_completion_order_check CHECK (completed_at >= created_at),
  CONSTRAINT generation_calls_idea_owner_fk FOREIGN KEY (idea_id, owner_id) REFERENCES ideas (id, owner_id),
  CONSTRAINT generation_calls_engine_version_fk FOREIGN KEY (engine_version_id) REFERENCES engine_versions (id),
  CONSTRAINT generation_calls_job_attempt_key UNIQUE (job_id, attempt),
  -- This composite target lets child rows prove their Idea, owner, and Engine Version match the call.
  CONSTRAINT generation_calls_identity_key UNIQUE (id, idea_id, owner_id, engine_version_id)
);

-- Job-scoped acceptance resolves one successful attempt.
CREATE UNIQUE INDEX generation_calls_job_success_idx ON generation_calls (job_id)
WHERE
  outcome = 'ok';

CREATE INDEX generation_calls_owner_id_idx ON generation_calls (owner_id);

CREATE INDEX generation_calls_idea_id_idx ON generation_calls (idea_id);

CREATE TABLE step_values (
  id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL,
  step_key text NOT NULL CHECK (step_key <> ''),
  generation_call_id uuid NOT NULL UNIQUE,
  value jsonb NOT NULL CHECK (jsonb_typeof(value) = 'object'),
  created_at timestamp with time zone NOT NULL,
  FOREIGN KEY (
    generation_call_id,
    idea_id,
    owner_id,
    engine_version_id
  ) REFERENCES generation_calls (id, idea_id, owner_id, engine_version_id),
  UNIQUE (idea_id, engine_version_id, step_key)
);

CREATE INDEX step_values_owner_id_idx ON step_values (owner_id);

CREATE TABLE manuscripts (
  id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL,
  accepted_generation_call_id uuid NOT NULL UNIQUE,
  value jsonb NOT NULL CHECK (jsonb_typeof(value) = 'object'),
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone NOT NULL,
  FOREIGN KEY (
    accepted_generation_call_id,
    idea_id,
    owner_id,
    engine_version_id
  ) REFERENCES generation_calls (id, idea_id, owner_id, engine_version_id)
);

CREATE INDEX manuscripts_owner_id_idx ON manuscripts (owner_id);

CREATE INDEX manuscripts_idea_id_idx ON manuscripts (idea_id);
