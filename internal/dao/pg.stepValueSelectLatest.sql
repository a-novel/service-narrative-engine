SELECT
  latest.*
FROM
  (
    SELECT DISTINCT
      ON (step_key) *
    FROM
      step_values
    WHERE
      idea_id = ?0
      AND NOT (step_key = ANY (?1::text[]))
    ORDER BY
      step_key,
      created_at DESC,
      id DESC
  ) AS latest
ORDER BY
  latest.step_key;
