-- The project's foreign key to its default instance refuses the delete of that instance.
DELETE FROM instances
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
