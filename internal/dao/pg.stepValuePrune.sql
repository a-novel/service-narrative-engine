DELETE FROM step_values
WHERE
  id IN (
    SELECT
      id
    FROM
      step_values
    WHERE
      idea_id = ?0
      AND step_key = ?1
    ORDER BY
      created_at DESC,
      id DESC
    OFFSET
      ?2
  );
