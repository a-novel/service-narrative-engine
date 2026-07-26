INSERT INTO
  ideas (
    id,
    owner_id,
    seed,
    story_type,
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
    'novel',
    'speculative',
    'The Answering Light',
    '2026-07-26T00:00:00Z',
    '2026-07-26T00:00:00Z'
  );

INSERT INTO
  generation_calls (
    job_id,
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
    '00000000-0000-0000-0000-000000000042',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000100',
    'openai',
    'resp_fixture',
    'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'fixture-model',
    'ok',
    '{"title":"The Answering Light","format":"prose","scenes":[]}',
    10,
    20,
    30,
    250,
    '2026-07-26T00:00:00Z',
    '2026-07-26T00:00:01Z'
  );

INSERT INTO
  step_values (
    id,
    owner_id,
    idea_id,
    engine_version_id,
    step_key,
    generation_job_id,
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
    '{"title":"The Answering Light","format":"prose","scenes":[]}',
    '2026-07-26T00:00:01Z'
  );

INSERT INTO
  manuscripts (
    id,
    owner_id,
    idea_id,
    accepted_generation_job_id,
    title,
    format,
    created_at,
    updated_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000204',
    '00000000-0000-0000-0000-000000000042',
    '00000000-0000-0000-0000-000000000201',
    '00000000-0000-0000-0000-000000000202',
    'The Answering Light',
    'prose',
    '2026-07-26T00:00:01Z',
    '2026-07-26T00:00:01Z'
  );

INSERT INTO
  manuscript_scenes (id, manuscript_id, ordinal, title)
VALUES
  (
    '00000000-0000-0000-0000-000000000205',
    '00000000-0000-0000-0000-000000000204',
    0,
    'The Reply'
  );

INSERT INTO
  manuscript_blocks (id, scene_id, ordinal, kind, text)
VALUES
  (
    '00000000-0000-0000-0000-000000000206',
    '00000000-0000-0000-0000-000000000205',
    0,
    'prose',
    'The second foghorn answered from below.'
  );
