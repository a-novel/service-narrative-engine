-- The kinds of generation the service runs through genai.
CREATE TYPE generation_kind AS ENUM('suggest', 'consolidate', 'check');

-- Where a call of a generation stands. A call waits as pending until genai admits it. A generation's own
-- status follows from its calls: it runs until every call settles.
CREATE TYPE generation_call_status AS ENUM(
  'pending',
  'running',
  'succeeded',
  'failed',
  'cancelled'
);

CREATE TABLE generations (
  -- A UUIDv7 sorts by creation time, so the primary key also orders a project's generations.
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  kind generation_kind NOT NULL,
  /* The request as the author sent it. */
  request jsonb NOT NULL,
  /* A digest of the request's kind, its variant and the payload of each call. A resent request plans
  the same payloads while the project holds still, so it finds its generation and resumes it. */
  request_key bytea NOT NULL,
  created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT generations_project_id_request_key_key UNIQUE (project_id, request_key)
);

-- Lists a project's generations newest first, and finds those past the latest 100.
CREATE INDEX generations_project_id_idx ON generations (project_id, id DESC);

CREATE TABLE generation_calls (
  generation_id uuid NOT NULL REFERENCES generations (id) ON DELETE CASCADE,
  /* The call's place in the generation's context plan, from 0. */
  position integer NOT NULL CHECK (position >= 0),
  /* What the call reads: the level and the revision of each part of its context, and the handles that
  map its output back. */
  context jsonb NOT NULL,
  /* The JSON body genai sends the provider, as text so its members keep their order. It stays until
  the call succeeds, so a call that waits, fails or is cancelled resubmits it as genai expects to
  replay it. */
  payload text,
  /* The genai generation that runs the call, once genai admits it. A call cancelled while it waited
  has none. */
  genai_id uuid,
  status generation_call_status NOT NULL DEFAULT 'pending',
  /* The call's output, mapped back through its handles. */
  result jsonb,
  /* Why a failed call failed. */
  error text CHECK (error <> ''),
  settled_at timestamp with time zone,
  PRIMARY KEY (generation_id, position),
  -- genai admits a call before it runs, a success drops its payload for its result, and only a
  -- failure carries a reason.
  CONSTRAINT generation_calls_status_check CHECK (
    (
      status <> 'pending'
      OR genai_id IS NULL
    )
    AND (
      status NOT IN ('running', 'succeeded')
      OR genai_id IS NOT NULL
    )
    AND (status IN ('pending', 'running')) = (settled_at IS NULL)
    AND (status = 'succeeded') = (result IS NOT NULL)
    AND (status = 'succeeded') = (payload IS NULL)
    AND (status = 'failed') = (error IS NOT NULL)
  )
);
