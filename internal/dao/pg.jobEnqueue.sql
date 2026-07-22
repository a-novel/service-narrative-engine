-- The no-op DO UPDATE exists so the statement returns a row on conflict; DO NOTHING returns none and
-- would force a second round-trip to fetch the winner. Which call created the row is decided by
-- comparing the returned id to the one the caller minted.
INSERT INTO
  jobs (
    id,
    kind,
    payload,
    owner_id,
    idempotency_key,
    request_fingerprint,
    max_attempts
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6)
ON CONFLICT (owner_id, kind, idempotency_key)
WHERE
  idempotency_key IS NOT NULL DO UPDATE
SET
  idempotency_key = jobs.idempotency_key
RETURNING
  *;
