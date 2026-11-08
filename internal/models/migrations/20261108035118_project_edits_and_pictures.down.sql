-- Drops every picture with the table. The stamps derive from earlier writes, so dropping them loses no
-- content.
DROP TABLE IF EXISTS project_pictures;

DROP TYPE IF EXISTS project_picture;

DROP INDEX IF EXISTS projects_owner_id_edited_at_idx;

CREATE INDEX projects_owner_id_idx ON projects (owner_id, id DESC);

ALTER TABLE projects
DROP COLUMN IF EXISTS background_updated_at,
DROP COLUMN IF EXISTS logo_updated_at,
DROP COLUMN IF EXISTS edited_at;
