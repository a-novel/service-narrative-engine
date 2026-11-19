SELECT
  *
FROM
  plugin_messages
WHERE
  plugin = ?0
  AND major = ?1
  AND minor = ?2
  AND patch = ?3
  AND prerelease = ?4
  AND language = ?5;
