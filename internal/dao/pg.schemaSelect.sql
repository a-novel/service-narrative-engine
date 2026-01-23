SELECT
  *
FROM
  schemas
WHERE
  -- If ID is provided, use it
  CASE
    WHEN ?0::uuid IS NOT NULL THEN id = ?0
    ELSE project_id = ?1
    AND module_id = ?2
    AND module_namespace = ?3
  END
ORDER BY
  created_at DESC
LIMIT
  1;
