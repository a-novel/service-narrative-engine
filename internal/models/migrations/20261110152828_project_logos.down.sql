-- Restores the previous schema. The backgrounds the up migration deleted do not come back. The restored media type
-- check refuses PNG, so the PNG logos are deleted first, and they do not come back either.
DELETE FROM project_logos
WHERE
  media_type = 'image/png';

CREATE TYPE project_picture AS ENUM('logo', 'background');

ALTER TABLE project_logos
RENAME TO project_pictures;

ALTER TABLE project_pictures
RENAME CONSTRAINT project_logos_project_id_fkey TO project_pictures_project_id_fkey;

ALTER TABLE project_pictures
RENAME CONSTRAINT project_logos_project_id_not_null TO project_pictures_project_id_not_null;

ALTER TABLE project_pictures
RENAME CONSTRAINT project_logos_data_not_null TO project_pictures_data_not_null;

ALTER TABLE project_pictures
RENAME CONSTRAINT project_logos_media_type_not_null TO project_pictures_media_type_not_null;

ALTER TABLE project_pictures
RENAME CONSTRAINT project_logos_updated_at_not_null TO project_pictures_updated_at_not_null;

-- Every remaining row is a logo, so the new column takes that value from a default the next statement drops.
ALTER TABLE project_pictures
DROP CONSTRAINT IF EXISTS project_logos_pkey,
DROP CONSTRAINT IF EXISTS project_logos_data_check,
DROP CONSTRAINT IF EXISTS project_logos_media_type_check,
ADD COLUMN picture project_picture NOT NULL DEFAULT 'logo',
ADD CONSTRAINT project_pictures_pkey PRIMARY KEY (project_id, picture),
ADD CONSTRAINT project_pictures_media_type_check CHECK (media_type IN ('image/webp', 'image/jpeg')),
ADD CONSTRAINT project_pictures_data_check CHECK (
  (
    picture = 'logo'
    AND octet_length(data) <= 262144
  )
  OR (
    picture = 'background'
    AND octet_length(data) <= 524288
  )
);

ALTER TABLE project_pictures
ALTER COLUMN picture
DROP DEFAULT;
