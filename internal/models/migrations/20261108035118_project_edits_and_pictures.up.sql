-- edited_at is when the project's content last changed: its own fields, its beats, scenes, layers and cells, and
-- their recaps. Every content write locks the project, and the lock stamps it, so a list orders the projects by
-- their last edit. Pictures are cosmetic, so their writes leave edited_at as it was. updated_at keeps the time
-- the project's own fields last changed.
--
-- The backfill takes the latest updated_at of the project and of its rows, and the index replaces the one
-- ordered by ID. No platform creates projects before this release, so the table is small enough to backfill
-- and index in plain statements.
ALTER TABLE projects
ADD COLUMN edited_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP;

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

-- A picture keeps its bytes and its version in its own row, so a picture write never writes the projects row.
-- A project read joins each picture's version through the primary key and never selects the bytes.
CREATE TABLE project_pictures (
  project_id uuid NOT NULL CONSTRAINT project_pictures_project_id_fkey REFERENCES projects (id) ON DELETE CASCADE,
  picture project_picture NOT NULL,
  /* The picture's encoded bytes, a WebP or a JPEG. */
  data bytea NOT NULL,
  /* The media type the picture is encoded in. */
  media_type text NOT NULL CONSTRAINT project_pictures_media_type_check CHECK (media_type IN ('image/webp', 'image/jpeg')),
  /* When the picture last changed. Picture URLs carry it as the picture's version. */
  updated_at timestamp with time zone NOT NULL,
  -- One picture of each kind per project: a logo and a background may coexist. A new picture replaces the
  -- stored one of its kind through this key.
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
