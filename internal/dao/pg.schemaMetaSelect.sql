SELECT
  s.id,
  s.project_id,
  s.owner,
  s.module_id,
  s.module_namespace,
  s.module_version,
  s.module_preversion,
  s.source,
  s.created_at,
  s.data IS NULL AS is_nil,
  NOT EXISTS (
    SELECT
      1
    FROM
      schemas s2
    WHERE
      s2.project_id = s.project_id
      AND s2.module_id = s.module_id
      AND s2.module_namespace = s.module_namespace
      AND s2.created_at > s.created_at
  ) AS is_latest
FROM
  schemas s
WHERE
  -- If ID is provided, use it
  CASE
    WHEN ?0::uuid IS NOT NULL THEN s.id = ?0
    ELSE s.project_id = ?1
    AND s.module_id = ?2
    AND s.module_namespace = ?3
  END
ORDER BY
  s.created_at DESC
LIMIT
  1;
