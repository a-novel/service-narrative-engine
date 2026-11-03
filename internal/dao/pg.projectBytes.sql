-- The project's own bytes stay in its row, beside the count of the rows it holds.
SELECT
  bytes + stored_bytes AS stored_bytes
FROM
  projects
WHERE
  id = ?0;
