-- The first key names the lock's purpose, project creation, so another advisory lock never shares it.
SELECT
  pg_advisory_xact_lock(1, hashtext (?0::text));
