DROP INDEX IF EXISTS manuscripts_history_idx;

DROP INDEX IF EXISTS step_values_engine_version_id_idx;

DROP INDEX IF EXISTS step_values_history_idx;

DROP INDEX IF EXISTS idea_versions_history_idx;

ALTER TABLE ideas
ADD COLUMN seed text,
ADD COLUMN genre text,
ADD COLUMN title text NOT NULL DEFAULT '',
ADD COLUMN updated_at timestamp with time zone;

WITH
  latest_idea_versions AS (
    SELECT DISTINCT
      ON (idea_id) idea_id,
      id,
      seed,
      genre,
      title,
      created_at
    FROM
      idea_versions
    ORDER BY
      idea_id,
      created_at DESC,
      id DESC
  )
UPDATE ideas
SET
  seed = latest_idea_versions.seed,
  genre = latest_idea_versions.genre,
  title = latest_idea_versions.title,
  updated_at = CASE
    WHEN latest_idea_versions.id = ideas.id
    AND latest_idea_versions.created_at = ideas.created_at THEN NULL
    ELSE latest_idea_versions.created_at
  END
FROM
  latest_idea_versions
WHERE
  latest_idea_versions.idea_id = ideas.id;

-- The versioned schema accepts partial Ideas, while the schema being restored does not.
UPDATE ideas
SET
  seed = 'unspecified'
WHERE
  seed = '';

UPDATE ideas
SET
  genre = 'unspecified'
WHERE
  genre = '';

ALTER TABLE ideas
ALTER COLUMN seed
SET NOT NULL,
ALTER COLUMN genre
SET NOT NULL;

ALTER TABLE ideas
ADD CONSTRAINT ideas_seed_check CHECK (seed <> '');

ALTER TABLE ideas
ADD CONSTRAINT ideas_genre_check CHECK (genre <> '');

DROP TABLE IF EXISTS idea_versions;

ALTER TABLE manuscripts
ADD COLUMN updated_at timestamp with time zone;

-- One row per Idea, Engine Version, and step is required by the restored schema.
WITH
  ranked_step_values AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          idea_id,
          engine_version_id,
          step_key
        ORDER BY
          created_at DESC,
          id DESC
      ) AS version_number
    FROM
      step_values
  )
DELETE FROM step_values
WHERE
  id IN (
    SELECT
      id
    FROM
      ranked_step_values
    WHERE
      version_number > 1
  );

ALTER TABLE step_values
ADD CONSTRAINT step_values_idea_id_engine_version_id_step_key_key UNIQUE (idea_id, engine_version_id, step_key);

CREATE INDEX manuscripts_idea_id_idx ON manuscripts (idea_id);
