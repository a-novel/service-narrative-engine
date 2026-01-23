SELECT
  version,
  preversion,
  created_at
FROM
  modules
WHERE
  id = ?0
  AND namespace = ?1
  -- Filter out preversions unless explicitly requested
  AND (
    preversion = ''
    OR ?4
  )
  -- Filter by specific version if provided
  AND (
    version = ?5
    OR ?5 = ''
  )
ORDER BY
  -- Sort by version descending (numeric-aware if versions are like 1.2.3)
  string_to_array(version, '.')::int[] DESC,
  -- Stable versions (empty preversion) first
  (preversion = '') DESC,
  -- For preversions: newest first
  created_at DESC
LIMIT
  ?2
OFFSET
  ?3;
