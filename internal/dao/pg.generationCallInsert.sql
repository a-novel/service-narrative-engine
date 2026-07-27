INSERT INTO
  generation_calls (
    job_id,
    attempt,
    owner_id,
    provider,
    model,
    input_tokens,
    output_tokens,
    created_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7)
RETURNING
  *;
