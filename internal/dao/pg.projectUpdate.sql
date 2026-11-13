-- RETURNING cannot join, so the outer query joins the logo version and checksum to the updated row.
WITH
  updated AS (
    UPDATE projects
    SET
      title = ?1,
      premise = ?2,
      language = ?3,
      tags = ?4,
      genres = ?5,
      updated_at = ?6
    WHERE
      id = ?0
    RETURNING
      *
  )
SELECT
  updated.*,
  logos.updated_at AS logo_updated_at,
  encode(logos.checksum, 'hex') AS logo_checksum
FROM
  updated
  LEFT JOIN project_logos AS logos ON logos.project_id = updated.id;
