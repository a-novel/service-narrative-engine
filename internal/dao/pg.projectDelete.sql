-- The outer query reads the logo as it stood before the delete, so it returns the checksum the project held.
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
  encode(logos.checksum, 'hex') AS logo_checksum
FROM
  deleted
  LEFT JOIN project_logos AS logos ON logos.project_id = deleted.id;
