CREATE TABLE idea_versions (
  id uuid PRIMARY KEY NOT NULL,
  idea_id uuid NOT NULL,
  seed text NOT NULL,
  genre text NOT NULL,
  title text NOT NULL DEFAULT '',
  created_at timestamp with time zone NOT NULL,
  CONSTRAINT idea_versions_idea_fk FOREIGN KEY (idea_id) REFERENCES ideas (id)
);

INSERT INTO
  idea_versions (id, idea_id, seed, genre, title, created_at)
SELECT
  id,
  id,
  seed,
  genre,
  title,
  COALESCE(updated_at, created_at)
FROM
  ideas;

ALTER TABLE ideas
DROP COLUMN seed,
DROP COLUMN genre,
DROP COLUMN title,
DROP COLUMN updated_at;

ALTER TABLE step_values
DROP CONSTRAINT step_values_idea_id_engine_version_id_step_key_key;

UPDATE manuscripts
SET
  created_at = COALESCE(updated_at, created_at);

ALTER TABLE manuscripts
DROP COLUMN updated_at;

DROP INDEX manuscripts_idea_id_idx;

CREATE INDEX idea_versions_history_idx ON idea_versions (idea_id, created_at DESC, id DESC);

CREATE INDEX step_values_history_idx ON step_values (idea_id, step_key, created_at DESC, id DESC);

CREATE INDEX step_values_engine_version_id_idx ON step_values (engine_version_id);

CREATE INDEX manuscripts_history_idx ON manuscripts (idea_id, created_at DESC, id DESC);

-- Bring existing histories under the same bound enforced for future writes.
WITH
  ranked_step_values AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          idea_id,
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
      version_number > 25
  );

WITH
  ranked_manuscripts AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          idea_id
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
