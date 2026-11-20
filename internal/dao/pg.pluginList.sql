-- The latest release of each plugin comes from the primary key alone. Every key sorts descending, so the key
-- read backward gives the order and nothing is sorted. Only those versions then read their translations, in one
-- read of the key each: the asked language where the version has it, English otherwise, and the first of its
-- other languages when it has neither.
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
      plugin DESC,
      major DESC,
      minor DESC,
      patch DESC
  )
SELECT
  latest.*,
  messages.title,
  messages.description,
  count(*) OVER () AS total
FROM
  latest
  JOIN LATERAL (
    SELECT
      title,
      description
    FROM
      plugin_messages
    WHERE
      (plugin, major, minor, patch, prerelease) = (
        latest.plugin,
        latest.major,
        latest.minor,
        latest.patch,
        latest.prerelease
      )
    ORDER BY
      language = ?0 DESC,
      language = 'en' DESC,
      language
    LIMIT
      1
  ) AS messages ON TRUE
ORDER BY
  latest.plugin
LIMIT
  ?1
OFFSET
  ?2;
