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
  created_at timestamp(0) with time zone NOT NULL,
  updated_at timestamp(0) with time zone NOT NULL,
  UNIQUE (id, owner_id)
);

CREATE INDEX ideas_owner_id_idx ON ideas (owner_id);

CREATE TABLE engine_versions (
  id uuid PRIMARY KEY NOT NULL,
  slug text NOT NULL CHECK (slug <> ''),
  version text NOT NULL CHECK (version <> ''),
  definition jsonb NOT NULL CHECK (jsonb_typeof(definition) = 'object'),
  content_hash text NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
  created_at timestamp(0) with time zone NOT NULL,
  UNIQUE (slug, version),
  UNIQUE (content_hash)
);

CREATE TABLE generation_calls (
  job_id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL REFERENCES engine_versions (id),
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
  created_at timestamp(0) with time zone NOT NULL,
  completed_at timestamp(0) with time zone NOT NULL,
  FOREIGN KEY (idea_id, owner_id) REFERENCES ideas (id, owner_id),
  UNIQUE (job_id, owner_id)
);

CREATE UNIQUE INDEX generation_calls_provider_call_id_idx ON generation_calls (provider_call_id)
WHERE
  provider_call_id IS NOT NULL;

CREATE INDEX generation_calls_owner_id_idx ON generation_calls (owner_id);

CREATE INDEX generation_calls_idea_id_idx ON generation_calls (idea_id);

CREATE TABLE step_values (
  id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  engine_version_id uuid NOT NULL REFERENCES engine_versions (id),
  step_key text NOT NULL CHECK (step_key <> ''),
  generation_job_id uuid NOT NULL UNIQUE,
  value jsonb NOT NULL CHECK (jsonb_typeof(value) = 'object'),
  created_at timestamp(0) with time zone NOT NULL,
  FOREIGN KEY (idea_id, owner_id) REFERENCES ideas (id, owner_id),
  FOREIGN KEY (generation_job_id, owner_id) REFERENCES generation_calls (job_id, owner_id),
  UNIQUE (idea_id, engine_version_id, step_key)
);

CREATE INDEX step_values_owner_id_idx ON step_values (owner_id);

CREATE TABLE manuscripts (
  id uuid PRIMARY KEY NOT NULL,
  owner_id uuid NOT NULL,
  idea_id uuid NOT NULL,
  accepted_generation_job_id uuid NOT NULL UNIQUE,
  title text NOT NULL CHECK (title <> ''),
  format text NOT NULL CHECK (format <> ''),
  created_at timestamp(0) with time zone NOT NULL,
  updated_at timestamp(0) with time zone NOT NULL,
  FOREIGN KEY (idea_id, owner_id) REFERENCES ideas (id, owner_id),
  FOREIGN KEY (accepted_generation_job_id, owner_id) REFERENCES generation_calls (job_id, owner_id)
);

CREATE INDEX manuscripts_owner_id_idx ON manuscripts (owner_id);

CREATE INDEX manuscripts_idea_id_idx ON manuscripts (idea_id);

CREATE TABLE manuscript_scenes (
  id uuid PRIMARY KEY NOT NULL,
  manuscript_id uuid NOT NULL REFERENCES manuscripts (id) ON DELETE CASCADE,
  ordinal integer NOT NULL CHECK (ordinal >= 0),
  title text NOT NULL CHECK (title <> ''),
  UNIQUE (manuscript_id, ordinal)
);

CREATE INDEX manuscript_scenes_manuscript_id_idx ON manuscript_scenes (manuscript_id);

CREATE TABLE manuscript_blocks (
  id uuid PRIMARY KEY NOT NULL,
  scene_id uuid NOT NULL REFERENCES manuscript_scenes (id) ON DELETE CASCADE,
  ordinal integer NOT NULL CHECK (ordinal >= 0),
  kind text NOT NULL CHECK (kind IN ('prose', 'dialogue', 'cue')),
  text text NOT NULL CHECK (text <> ''),
  UNIQUE (scene_id, ordinal)
);

CREATE INDEX manuscript_blocks_scene_id_idx ON manuscript_blocks (scene_id);

INSERT INTO
  engine_versions (
    id,
    slug,
    version,
    definition,
    content_hash,
    created_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000100',
    'walking-skeleton',
    '0.0.1',
    '{
      "kind": "project",
      "steps": [
        {
          "key": "manuscript",
          "promptTemplate": "Turn the idea into a concise prose manuscript proposal.",
          "outputSchema": {
            "$schema": "https://json-schema.org/draft/2020-12/schema",
            "type": "object",
            "additionalProperties": false,
            "required": ["title", "format", "scenes"],
            "properties": {
              "title": {
                "type": "string",
                "minLength": 1
              },
              "format": {
                "const": "prose"
              },
              "scenes": {
                "type": "array",
                "minItems": 1,
                "items": {
                  "type": "object",
                  "additionalProperties": false,
                  "required": ["title", "blocks"],
                  "properties": {
                    "title": {
                      "type": "string",
                      "minLength": 1
                    },
                    "blocks": {
                      "type": "array",
                      "minItems": 1,
                      "items": {
                        "type": "object",
                        "additionalProperties": false,
                        "required": ["kind", "text"],
                        "properties": {
                          "kind": {
                            "enum": ["prose", "dialogue", "cue"]
                          },
                          "text": {
                            "type": "string",
                            "minLength": 1
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      ]
    }'::jsonb,
    '42a95e527dda098d5ed17109a0106c3a29cce826e78a75c5b4fea102b143d30c',
    '2026-07-26T00:00:00Z'
  );
