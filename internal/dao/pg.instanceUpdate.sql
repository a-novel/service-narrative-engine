-- A rename leaves updated_at as it was. A content write names that value, and a new name changes nothing the
-- write would replace, so it must not refuse the write.
UPDATE instances
SET
  name = ?2
WHERE
  project_id = ?0
  AND id = ?1
RETURNING
  id,
  project_id,
  plugin,
  version,
  name,
  created_at,
  updated_at;
