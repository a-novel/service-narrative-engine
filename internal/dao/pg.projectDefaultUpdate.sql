-- The foreign key carries the project, so the write fails on an instance the project does not hold. The default is
-- a field of the project, so the write stamps updated_at, which the fields drift reads. RETURNING cannot join, so the
-- outer query joins the logo checksum to the updated row.
WITH
  updated AS (
    UPDATE projects
    SET
      default_instance_id = ?1,
      updated_at = ?2
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
