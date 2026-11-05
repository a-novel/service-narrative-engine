-- The summaries are columns each write of a cell keeps, so a page reads neither its content nor its recap.
-- The page follows the index on the project and the layer.
SELECT
  scene_id,
  layer_id,
  updated_at,
  content_empty,
  content_size,
  content_speakers,
  recap_names,
  recap_condensed_size,
  recap_compact_size,
  recap_essentials_size,
  recap_cell_updated_at,
  count(*) OVER () AS total
FROM
  cells
WHERE
  project_id = ?0
ORDER BY
  layer_id,
  scene_id
LIMIT
  ?1
OFFSET
  ?2;
