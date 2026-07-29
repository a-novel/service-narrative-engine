-- Rollback requires every existing Idea to have a non-empty seed and genre.
ALTER TABLE ideas
ADD CONSTRAINT ideas_seed_check CHECK (seed <> '');

ALTER TABLE ideas
ADD CONSTRAINT ideas_genre_check CHECK (genre <> '');
