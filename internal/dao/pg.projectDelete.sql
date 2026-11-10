-- The outer query reads the logo as it stood before the delete, so it returns the version the project held.
WITH
  deleted AS (
    DELETE FROM projects
    WHERE
      id = ?0
    RETURNING
      *
  )
SELECT
  deleted.*,
  logos.updated_at AS logo_updated_at
FROM
  deleted
  LEFT JOIN project_logos AS logos ON logos.project_id = deleted.id;
