-- Every summary derives from the documents it measures, so dropping the columns loses no data. The columns
-- depend on the functions, so they go first.
ALTER TABLE cells
DROP COLUMN IF EXISTS recap_essentials_size,
DROP COLUMN IF EXISTS recap_compact_size,
DROP COLUMN IF EXISTS recap_condensed_size,
DROP COLUMN IF EXISTS recap_names,
DROP COLUMN IF EXISTS content_speakers,
DROP COLUMN IF EXISTS content_size,
DROP COLUMN IF EXISTS content_empty;

DROP FUNCTION IF EXISTS cell_content_speakers (jsonb);

DROP FUNCTION IF EXISTS cell_content_size (jsonb);
