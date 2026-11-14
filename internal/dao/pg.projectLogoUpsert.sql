-- The logo and its checksum change in one row, and the project returns with the written checksum.
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
      checksum
  )
SELECT
  projects.*,
  encode(stored.checksum, 'hex') AS logo_checksum
FROM
  projects
  CROSS JOIN stored
WHERE
  projects.id = ?0;
