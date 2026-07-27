INSERT INTO
  engine_versions (id, kind, slug, version, definition, created_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000100',
    'project',
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
    '2026-07-26T00:00:00.123456Z'
  );

INSERT INTO
  ideas (
    id,
    owner_id,
    seed,
    genre,
    title,
    created_at,
    updated_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000042',
    'A lighthouse keeper hears a second foghorn answer from beneath the sea.',
    'speculative',
    'The Answering Light',
    '2026-07-26T00:00:00.123456Z',
    '2026-07-26T00:00:00.123456Z'
  );

INSERT INTO
  generation_calls (
    id,
    job_id,
    attempt,
    owner_id,
    idea_id,
    engine_version_id,
    provider,
    provider_call_id,
    request_hash,
    model,
    outcome,
    raw_output,
    input_tokens,
    output_tokens,
    total_tokens,
    latency_ms,
    created_at,
    completed_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000202',
    '00000000-0000-0000-0000-000000000210',
    1,
    '00000000-0000-0000-0000-000000000042',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000100',
    'openai',
    'resp_fixture',
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'fixture-model',
    'ok',
    '{"title":"The Answering Light","format":"prose","scenes":[{"title":"The Reply","blocks":[{"kind":"prose","text":"The buried foghorn answers."}]}]}',
    10,
    20,
    30,
    250,
    '2026-07-26T00:00:00.123456Z',
    '2026-07-26T00:00:00.373456Z'
  );

INSERT INTO
  step_values (
    id,
    owner_id,
    idea_id,
    engine_version_id,
    step_key,
    generation_call_id,
    value,
    created_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000203',
    '00000000-0000-0000-0000-000000000042',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000100',
    'manuscript',
    '00000000-0000-0000-0000-000000000202',
    '{"title":"The Answering Light","format":"prose","scenes":[{"title":"The Reply","blocks":[{"kind":"prose","text":"The buried foghorn answers."}]}]}',
    '2026-07-26T00:00:00.373456Z'
  );

INSERT INTO
  manuscripts (
    id,
    owner_id,
    idea_id,
    engine_version_id,
    accepted_generation_call_id,
    value,
    created_at,
    updated_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000204',
    '00000000-0000-0000-0000-000000000042',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000100',
    '00000000-0000-0000-0000-000000000202',
    '{"title":"The Answering Light","format":"prose","scenes":[{"title":"The Reply","blocks":[{"kind":"prose","text":"The buried foghorn answers."}]}]}',
    '2026-07-26T00:00:00.373456Z',
    '2026-07-26T00:00:00.373456Z'
  );
