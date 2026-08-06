ALTER TABLE ideas
RENAME TO projects;

ALTER TABLE projects
RENAME CONSTRAINT ideas_pkey TO projects_pkey;

CREATE TABLE idea_versions (
  id uuid PRIMARY KEY NOT NULL,
  project_id uuid NOT NULL,
  seed text NOT NULL,
  genre text NOT NULL,
  title text NOT NULL DEFAULT '',
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT idea_versions_project_fk FOREIGN KEY (project_id) REFERENCES projects (id)
);

INSERT INTO
  idea_versions (id, project_id, seed, genre, title, created_at)
SELECT
  id,
  id,
  seed,
  genre,
  title,
  COALESCE(updated_at, created_at)
FROM
  projects;

ALTER TABLE projects
DROP COLUMN seed,
DROP COLUMN genre,
DROP COLUMN title,
DROP COLUMN updated_at;

-- Engine qualification owns definition shape; runtime storage requires only valid JSONB.
ALTER TABLE engine_versions
DROP CONSTRAINT engine_versions_definition_check;

ALTER TABLE step_values
DROP CONSTRAINT step_values_idea_id_engine_version_id_step_key_key,
DROP CONSTRAINT step_values_engine_version_fk,
DROP CONSTRAINT step_values_value_check;

ALTER TABLE step_values
RENAME COLUMN idea_id TO project_id;

ALTER TABLE step_values
RENAME COLUMN step_key TO key;

ALTER TABLE step_values
DROP COLUMN engine_version_id;

ALTER TABLE step_values
RENAME CONSTRAINT step_values_idea_fk TO step_values_project_fk;

ALTER TABLE step_values
RENAME CONSTRAINT step_values_step_key_check TO step_values_key_check;

UPDATE manuscripts
SET
  created_at = COALESCE(updated_at, created_at);

ALTER TABLE manuscripts
DROP COLUMN updated_at;

DROP INDEX manuscripts_idea_id_idx;

ALTER TABLE manuscripts
RENAME COLUMN idea_id TO project_id;

ALTER TABLE manuscripts
RENAME CONSTRAINT manuscripts_idea_fk TO manuscripts_project_fk;

CREATE INDEX idea_versions_history_idx ON idea_versions (project_id, created_at DESC, id DESC);

CREATE INDEX step_values_history_idx ON step_values (project_id, key, created_at DESC, id DESC);

CREATE INDEX manuscripts_history_idx ON manuscripts (project_id, created_at DESC, id DESC);

-- Bring existing histories under the same bound enforced for future writes.
WITH
  ranked_idea_versions AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          project_id
        ORDER BY
          created_at DESC,
          id DESC
      ) AS version_number
    FROM
      idea_versions
  )
DELETE FROM idea_versions
WHERE
  id IN (
    SELECT
      id
    FROM
      ranked_idea_versions
    WHERE
      version_number > 25
  );

WITH
  ranked_step_values AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          project_id,
          key
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
      version_number > 25
  );

WITH
  ranked_manuscripts AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          project_id
        ORDER BY
          created_at DESC,
          id DESC
      ) AS version_number
    FROM
      manuscripts
  )
DELETE FROM manuscripts
WHERE
  id IN (
    SELECT
      id
    FROM
      ranked_manuscripts
    WHERE
      version_number > 25
  );
