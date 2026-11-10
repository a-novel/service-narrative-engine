-- A project's tags and genres are the author's labels, in order. The service validates each label, and the checks cap
-- the lists. Every row starts empty, and the table is small enough for the checks to scan it under the ALTER's lock.
ALTER TABLE projects
ADD COLUMN tags text[] NOT NULL DEFAULT '{}' CONSTRAINT projects_tags_check CHECK (cardinality(tags) <= 10),
ADD COLUMN genres text[] NOT NULL DEFAULT '{}' CONSTRAINT projects_genres_check CHECK (cardinality(genres) <= 10);
