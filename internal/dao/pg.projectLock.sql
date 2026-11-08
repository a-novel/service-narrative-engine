-- The update locks the row until the transaction ends, so the project's writes run one at a time, and stamps
-- the edit they make.
UPDATE projects
SET
  edited_at = ?2
WHERE
  id = ?0
  AND owner_id = ?1;
