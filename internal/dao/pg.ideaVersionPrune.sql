DELETE FROM idea_versions
WHERE
  id IN (
    SELECT
      id
    FROM
      idea_versions
    WHERE
      project_id = ?0
    ORDER BY
      created_at DESC,
      id DESC
    OFFSET
      ?1
  );
