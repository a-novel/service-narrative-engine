-- A version and its translations are stored by one statement, so the registry holds both or neither. Each
-- translation takes the key of the version stored with it.
WITH
  version AS (
    INSERT INTO
      plugin_versions (
        plugin,
        major,
        minor,
        patch,
        prerelease,
        schema,
        exports,
        migration,
        ui,
        proof,
        api_key_id,
        published_at
      )
    VALUES
      (
        ?0,
        ?1,
        ?2,
        ?3,
        ?4,
        ?5::jsonb,
        ?6::jsonb,
        ?7,
        ?8,
        ?9,
        ?10,
        ?11
      )
    RETURNING
      plugin,
      major,
      minor,
      patch,
      prerelease
  )
INSERT INTO
  plugin_messages (
    plugin,
    major,
    minor,
    patch,
    prerelease,
    language,
    title,
    description,
    messages
  )
SELECT
  version.plugin,
  version.major,
  version.minor,
  version.patch,
  version.prerelease,
  data.language,
  data.title,
  data.description,
  data.messages::jsonb
FROM
  version,
  unnest(
    ARRAY[?12]::text[],
    ARRAY[?13]::text[],
    ARRAY[?14]::text[],
    ARRAY[?15]::text[]
  ) AS data (language, title, description, messages);
