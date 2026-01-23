-- Delete all schemas associated with this project
DELETE FROM schemas
WHERE
  project_id = ?0;

-- Delete the project and return it
DELETE FROM projects
WHERE
  id = ?0
RETURNING
  *;
