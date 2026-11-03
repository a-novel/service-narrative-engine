-- A fixture runs after its matching migration and seeds data for the next migration. The recaps give the
-- byte counts JSON documents to measure beside the null ones.
UPDATE cells
SET
  recap = '{"condensed": "", "compact": "", "essentials": "", "names": []}',
  recap_cell_updated_at = updated_at
WHERE
  scene_id = '00000000-0000-0000-0000-000000000011'
  AND layer_id = '00000000-0000-0000-0000-000000000021';

UPDATE beats
SET
  recap = '{"condensed": "", "compact": "", "essentials": "", "names": []}',
  recap_beat_updated_at = updated_at,
  recap_sources = '[]'
WHERE
  id = '00000000-0000-0000-0000-000000000011';
