-- The page takes the scenes from the offset while their bytes, summed in order, stay within the bound, and
-- always its first scene. The sums read the bytes column alone, so only the scenes of the page load their
-- text.
WITH
  ranked AS (
    SELECT
      scenes.id,
      scenes.bytes,
      row_number() OVER (
        ORDER BY
          beats.position,
          scenes.position
      ) AS rank,
      count(*) OVER () AS total
    FROM
      scenes
      JOIN beats ON beats.id = scenes.beat_id
    WHERE
      scenes.project_id = ?0
    ORDER BY
      beats.position,
      scenes.position
    LIMIT
      ?1
    OFFSET
      ?2
  ),
  page AS (
    SELECT
      id,
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
  scenes.*,
  page.total
FROM
  page
  JOIN scenes ON scenes.id = page.id
WHERE
  page.rank = ?2 + 1
  OR page.bytes <= ?3
ORDER BY
  page.rank;
