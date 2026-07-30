-- The older schema cannot represent partial Ideas. Keep rollback deterministic
-- by normalizing fields that were allowed to become empty after this migration.
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
ADD CONSTRAINT ideas_seed_check CHECK (seed <> '');

ALTER TABLE ideas
ADD CONSTRAINT ideas_genre_check CHECK (genre <> '');
