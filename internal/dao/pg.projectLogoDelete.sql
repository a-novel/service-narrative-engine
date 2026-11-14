-- The logo and its checksum go in one row, so the project returns with no logo checksum.
WITH
  deleted AS (
    DELETE FROM project_logos
    WHERE
      project_id = ?0
  )
SELECT
  projects.*
FROM
  projects
WHERE
  projects.id = ?0;
