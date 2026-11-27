-- The foreign key carries the project, so the write fails on an instance the project does not hold. RETURNING
-- cannot join, so the outer query joins the logo checksum to the updated row.
WITH
  updated AS (
    UPDATE projects
    SET
      default_instance_id = ?1
    WHERE
      id = ?0
    RETURNING
      *
  )
SELECT
  updated.*,
  encode(logos.checksum, 'hex') AS logo_checksum
FROM
  updated
  LEFT JOIN project_logos AS logos ON logos.project_id = updated.id;
