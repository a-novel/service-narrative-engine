-- The checksum derives from the logo's bytes, so dropping it loses no data.
ALTER TABLE project_logos
DROP COLUMN IF EXISTS checksum;
