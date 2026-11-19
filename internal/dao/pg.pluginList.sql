-- The latest release of each plugin comes from the primary key alone. Only those versions then read their
-- translations: the asked language where the version has it, English otherwise, which every version holds.
WITH
  latest AS (
    SELECT DISTINCT
      ON (plugin) plugin,
      major,
      minor,
      patch,
      prerelease
    FROM
      plugin_versions
    WHERE
      prerelease = ''
    ORDER BY
      plugin,
      major DESC,
      minor DESC,
      patch DESC
  )
SELECT
  latest.*,
  coalesce(asked.title, english.title) AS title,
  coalesce(asked.description, english.description) AS description,
  count(*) OVER () AS total
FROM
  latest
  JOIN plugin_messages AS english ON (
    english.plugin,
    english.major,
    english.minor,
    english.patch,
    english.prerelease,
    english.language
  ) = (
    latest.plugin,
    latest.major,
    latest.minor,
    latest.patch,
    latest.prerelease,
    'en'
  )
  LEFT JOIN plugin_messages AS asked ON (
    asked.plugin,
    asked.major,
    asked.minor,
    asked.patch,
    asked.prerelease,
    asked.language
  ) = (
    latest.plugin,
    latest.major,
    latest.minor,
    latest.patch,
    latest.prerelease,
    ?0
  )
ORDER BY
  latest.plugin
LIMIT
  ?1
OFFSET
  ?2;
