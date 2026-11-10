-- Drops every project's tags and genres with their columns, along with their checks.
ALTER TABLE projects
DROP COLUMN IF EXISTS genres,
DROP COLUMN IF EXISTS tags;
