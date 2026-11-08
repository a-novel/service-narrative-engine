-- edited_at is when anything in the project last changed: its own fields, its beats, scenes, layers and cells,
-- their recaps and its pictures. Every write locks the project, and the lock stamps it, so a list orders the
-- projects by their last edit. updated_at keeps the time the project's own fields last changed.
--
-- The backfill takes the latest updated_at of the project and of its rows, and the index replaces the one
-- ordered by ID. No platform creates projects before this release, so the table is small enough to backfill
-- and index in plain statements.
ALTER TABLE projects
ADD COLUMN edited_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
/* When the logo last changed, null without a logo. Picture URLs carry it as the logo's version. */
ADD COLUMN logo_updated_at timestamp with time zone,
/* When the background last changed, null without a background. Picture URLs carry it as its version. */
ADD COLUMN background_updated_at timestamp with time zone;

UPDATE projects
SET
  edited_at = GREATEST(
    updated_at,
    (
      SELECT
        max(updated_at)
      FROM
        beats
      WHERE
        beats.project_id = projects.id
    ),
    (
      SELECT
        max(updated_at)
      FROM
        scenes
      WHERE
        scenes.project_id = projects.id
    ),
    (
      SELECT
        max(updated_at)
      FROM
        layers
      WHERE
        layers.project_id = projects.id
    ),
    (
      SELECT
        max(updated_at)
      FROM
        cells
      WHERE
        cells.project_id = projects.id
    )
  );

DROP INDEX projects_owner_id_idx;

-- Lists a user's projects most recently edited first.
CREATE INDEX projects_owner_id_edited_at_idx ON projects (owner_id, edited_at DESC, id DESC);

-- The pictures a project may hold, one of each.
CREATE TYPE project_picture AS ENUM('logo', 'background');

-- The bytes stay out of the projects row, which every project read selects whole, so a list never reads them.
-- The versions stay in the projects row, which a picture write updates in the statement that writes the bytes.
CREATE TABLE project_pictures (
  project_id uuid NOT NULL CONSTRAINT project_pictures_project_id_fkey REFERENCES projects (id) ON DELETE CASCADE,
  picture project_picture NOT NULL,
  /* The picture's encoded bytes, a WebP or a JPEG. */
  data bytea NOT NULL,
  /* The media type the picture is encoded in. */
  media_type text NOT NULL CONSTRAINT project_pictures_media_type_check CHECK (media_type IN ('image/webp', 'image/jpeg')),
  PRIMARY KEY (project_id, picture),
  -- A logo holds 256 KiB at most, a background 512 KiB.
  CONSTRAINT project_pictures_data_check CHECK (
    (
      picture = 'logo'
      AND octet_length(data) <= 262144
    )
    OR (
      picture = 'background'
      AND octet_length(data) <= 524288
    )
  )
);
