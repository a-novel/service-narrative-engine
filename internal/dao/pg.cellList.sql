-- The page takes the cells from the offset while their bytes, summed in order, stay within the bound, and
-- always its first cell. The sums read the bytes column alone, so only the cells of the page load their
-- documents.
WITH
  ranked AS (
    SELECT
      cells.scene_id,
      cells.layer_id,
      cells.bytes,
      row_number() OVER (
        ORDER BY
          beats.position,
          scenes.position,
          layers.position
      ) AS rank,
      count(*) OVER () AS total
    FROM
      cells
      JOIN scenes ON scenes.id = cells.scene_id
      JOIN beats ON beats.id = scenes.beat_id
      JOIN layers ON layers.id = cells.layer_id
    WHERE
      cells.project_id = ?0
    ORDER BY
      beats.position,
      scenes.position,
      layers.position
    LIMIT
      ?1
    OFFSET
      ?2
  ),
  page AS (
    SELECT
      scene_id,
      layer_id,
      rank,
      total,
      sum(bytes) OVER (
        ORDER BY
          rank
      ) AS bytes
    FROM
      ranked
  )
SELECT
  cells.*,
  page.total
FROM
  page
  JOIN cells ON cells.scene_id = page.scene_id
  AND cells.layer_id = page.layer_id
WHERE
  page.rank = ?2 + 1
  OR page.bytes <= ?3
ORDER BY
  page.rank;
