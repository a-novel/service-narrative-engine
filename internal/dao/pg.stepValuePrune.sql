DELETE FROM step_values
WHERE
  id IN (
    SELECT
      id
    FROM
      step_values
    WHERE
      project_id = ?0
      AND key = ?1
    ORDER BY
      created_at DESC,
      id DESC
    OFFSET
      ?2
  );
