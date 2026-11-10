-- The logo and its version change in one row, and the project returns with the written version.
WITH
  stored AS (
    INSERT INTO
      project_logos (project_id, data, media_type, updated_at)
    VALUES
      (?0, ?1, ?2, ?3)
    ON CONFLICT (project_id) DO UPDATE
    SET
      data = EXCLUDED.data,
      media_type = EXCLUDED.media_type,
      updated_at = EXCLUDED.updated_at
    RETURNING
      updated_at
  )
SELECT
  projects.*,
  stored.updated_at AS logo_updated_at
FROM
  projects
  CROSS JOIN stored
WHERE
  projects.id = ?0;
