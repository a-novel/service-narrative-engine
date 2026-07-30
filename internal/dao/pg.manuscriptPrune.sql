DELETE FROM manuscripts
WHERE
  id IN (
    SELECT
      id
    FROM
      manuscripts
    WHERE
      idea_id = ?0
    ORDER BY
      created_at DESC,
      id DESC
    OFFSET
      ?1
  );
