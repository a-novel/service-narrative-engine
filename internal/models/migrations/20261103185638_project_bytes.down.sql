-- Every count derives from the text it measures, so dropping the columns loses no data.
ALTER TABLE projects
DROP COLUMN IF EXISTS stored_bytes,
DROP COLUMN IF EXISTS bytes;

ALTER TABLE layers
DROP COLUMN IF EXISTS bytes;

ALTER TABLE scenes
DROP COLUMN IF EXISTS bytes;

ALTER TABLE beats
DROP COLUMN IF EXISTS bytes;

ALTER TABLE cells
DROP COLUMN IF EXISTS bytes;
