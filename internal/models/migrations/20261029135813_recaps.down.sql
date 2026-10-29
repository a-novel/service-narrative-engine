-- Dropping the columns loses every recap.
ALTER TABLE beats
DROP COLUMN IF EXISTS recap_sources,
DROP COLUMN IF EXISTS recap_beat_updated_at,
DROP COLUMN IF EXISTS recap;

ALTER TABLE cells
DROP COLUMN IF EXISTS recap_cell_updated_at,
DROP COLUMN IF EXISTS recap;
