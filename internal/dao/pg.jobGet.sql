-- The owner is part of the predicate rather than a check on the returned row. A caller that omits it
-- fails to scan instead of returning someone else's job, and "no such job" and "not your job"
-- collapse into one no-rows result, so the response cannot be used to probe for ids.
SELECT
  *
FROM
  jobs
WHERE
  id = ?0
  AND owner_id = ?1;
