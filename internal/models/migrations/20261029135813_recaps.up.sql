-- A recap condenses a cell or a beat for later contexts, as a document that follows the consolidate
-- output schema. Its stamps name the versions of its sources it read, and it is fresh while they still
-- hold those versions. Writing a recap leaves its source's updated_at unchanged.
ALTER TABLE cells
ADD COLUMN recap jsonb,
ADD COLUMN recap_cell_updated_at timestamp with time zone;

-- A beat's recap also lists, for each of its scenes in order, the updated_at of the manuscript cell it
-- read, as a JSON array of {sceneID, cellUpdatedAt}.
ALTER TABLE beats
ADD COLUMN recap jsonb,
ADD COLUMN recap_beat_updated_at timestamp with time zone,
ADD COLUMN recap_sources jsonb;
