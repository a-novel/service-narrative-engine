-- A fixture runs after its matching migration and seeds data for the next migration. The fixture
-- generation runs two calls on the fixture project: the first succeeded, and the second waits for genai.
INSERT INTO
  generations (id, project_id, kind, request, request_key)
VALUES
  (
    '00000000-0000-0000-0000-000000000021',
    '00000000-0000-0000-0000-000000000001',
    'suggest',
    '{"targets": []}',
    '\x00'
  );

INSERT INTO
  generation_calls (
    generation_id,
    position,
    context,
    payload,
    genai_id,
    status,
    result,
    settled_at
  )
VALUES
  (
    '00000000-0000-0000-0000-000000000021',
    0,
    '{}',
    NULL,
    '00000000-0000-0000-0000-000000000022',
    'succeeded',
    '{"manuscript": [], "text": []}',
    CURRENT_TIMESTAMP
  ),
  (
    '00000000-0000-0000-0000-000000000021',
    1,
    '{}',
    '\x7b7d',
    NULL,
    'pending',
    NULL,
    NULL
  );
