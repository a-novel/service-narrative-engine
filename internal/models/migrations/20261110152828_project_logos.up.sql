-- A project now holds one picture, its logo, so the table no longer names which picture a row is. Backgrounds are
-- deleted first because the new key allows one row per project. A logo takes PNG as well, since Safari cannot encode
-- WebP and JPEG loses transparency. The table holds at most two rows per project, so plain statements fit it.
DELETE FROM project_pictures
WHERE
  picture = 'background';

ALTER TABLE project_pictures
RENAME TO project_logos;

ALTER TABLE project_logos
DROP CONSTRAINT project_pictures_pkey,
DROP CONSTRAINT project_pictures_data_check,
DROP CONSTRAINT project_pictures_media_type_check,
DROP COLUMN picture,
ADD CONSTRAINT project_logos_pkey PRIMARY KEY (project_id),
ADD CONSTRAINT project_logos_data_check CHECK (octet_length(data) <= 262144),
ADD CONSTRAINT project_logos_media_type_check CHECK (
  media_type IN ('image/webp', 'image/jpeg', 'image/png')
);

ALTER TABLE project_logos
RENAME CONSTRAINT project_pictures_project_id_fkey TO project_logos_project_id_fkey;

ALTER TABLE project_logos
RENAME CONSTRAINT project_pictures_project_id_not_null TO project_logos_project_id_not_null;

ALTER TABLE project_logos
RENAME CONSTRAINT project_pictures_data_not_null TO project_logos_data_not_null;

ALTER TABLE project_logos
RENAME CONSTRAINT project_pictures_media_type_not_null TO project_logos_media_type_not_null;

ALTER TABLE project_logos
RENAME CONSTRAINT project_pictures_updated_at_not_null TO project_logos_updated_at_not_null;

DROP TYPE project_picture;
