SELECT
  *
FROM
  modules
WHERE
  id = ?0
  AND namespace = ?1
  -- Either select the target version or the latest stable version (no preversion)
  AND CASE
    WHEN ?2 = '' THEN preversion = ''
    ELSE version = ?2
    AND preversion = ?3
  END
ORDER BY
  version DESC
LIMIT
  1;
