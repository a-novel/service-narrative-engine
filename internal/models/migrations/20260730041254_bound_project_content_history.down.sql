DROP INDEX IF EXISTS manuscripts_history_idx;

DROP INDEX IF EXISTS step_values_history_idx;

DROP INDEX IF EXISTS idea_versions_history_idx;

-- Definitions outside the restored object-only contract have no legacy representation.
DELETE FROM engine_versions
WHERE
  jsonb_typeof(definition) <> 'object';

ALTER TABLE engine_versions
ADD CONSTRAINT engine_versions_definition_check CHECK (jsonb_typeof(definition) = 'object');

ALTER TABLE projects
ADD COLUMN seed text,
ADD COLUMN genre text,
ADD COLUMN title text NOT NULL DEFAULT '',
ADD COLUMN updated_at timestamp with time zone;

WITH
  latest_idea_versions AS (
    SELECT DISTINCT
      ON (project_id) project_id,
      id,
      seed,
      genre,
      title,
      created_at
    FROM
      idea_versions
    ORDER BY
      project_id,
      created_at DESC,
      id DESC
  )
UPDATE projects
SET
  seed = latest_idea_versions.seed,
  genre = latest_idea_versions.genre,
  title = latest_idea_versions.title,
  updated_at = CASE
    WHEN latest_idea_versions.id = projects.id
    AND latest_idea_versions.created_at = projects.created_at THEN NULL
    ELSE latest_idea_versions.created_at
  END
FROM
  latest_idea_versions
WHERE
  latest_idea_versions.project_id = projects.id;

-- The versioned schema accepts partial Ideas, while the schema being restored does not.
UPDATE projects
SET
  seed = 'unspecified'
WHERE
  seed = '';

UPDATE projects
SET
  genre = 'unspecified'
WHERE
  genre = '';

ALTER TABLE projects
ALTER COLUMN seed
SET NOT NULL,
ALTER COLUMN genre
SET NOT NULL;

ALTER TABLE projects
ADD CONSTRAINT ideas_seed_check CHECK (seed <> '');

ALTER TABLE projects
ADD CONSTRAINT ideas_genre_check CHECK (genre <> '');

DROP TABLE IF EXISTS idea_versions;

ALTER TABLE step_values
ADD COLUMN engine_version_id uuid;

UPDATE step_values
SET
  engine_version_id = (
    SELECT
      id
    FROM
      engine_versions
    ORDER BY
      created_at DESC,
      id DESC
    LIMIT
      1
  );

-- Rows that cannot satisfy the restored Engine foreign key have no representation there.
DELETE FROM step_values
WHERE
  engine_version_id IS NULL
  OR jsonb_typeof(value) <> 'object';

ALTER TABLE step_values
ALTER COLUMN engine_version_id
SET NOT NULL;

ALTER TABLE step_values
RENAME COLUMN project_id TO idea_id;

ALTER TABLE step_values
RENAME COLUMN key TO step_key;

ALTER TABLE step_values
RENAME CONSTRAINT step_values_project_fk TO step_values_idea_fk;

ALTER TABLE step_values
RENAME CONSTRAINT step_values_key_check TO step_values_step_key_check;

ALTER TABLE step_values
ADD CONSTRAINT step_values_engine_version_fk FOREIGN KEY (engine_version_id) REFERENCES engine_versions (id);

ALTER TABLE step_values
ADD CONSTRAINT step_values_value_check CHECK (jsonb_typeof(value) = 'object');

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

ALTER TABLE manuscripts
RENAME COLUMN project_id TO idea_id;

ALTER TABLE manuscripts
RENAME CONSTRAINT manuscripts_project_fk TO manuscripts_idea_fk;

ALTER TABLE manuscripts
ADD COLUMN updated_at timestamp with time zone;

CREATE INDEX manuscripts_idea_id_idx ON manuscripts (idea_id);

ALTER TABLE projects
RENAME TO ideas;

ALTER TABLE ideas
RENAME CONSTRAINT projects_pkey TO ideas_pkey;

ALTER TABLE ideas
RENAME CONSTRAINT projects_seed_not_null TO ideas_seed_not_null;

ALTER TABLE ideas
RENAME CONSTRAINT projects_genre_not_null TO ideas_genre_not_null;

ALTER TABLE ideas
RENAME CONSTRAINT projects_title_not_null TO ideas_title_not_null;
