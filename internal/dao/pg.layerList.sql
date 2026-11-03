-- The page takes the layers from the offset while their bytes, summed in order, stay within the bound, and
-- always its first layer. The sums read the bytes column alone, so only the layers of the page load their
-- text.
WITH
  ranked AS (
    SELECT
      id,
      bytes,
      row_number() OVER (
        ORDER BY
          position
      ) AS rank,
      count(*) OVER () AS total
    FROM
      layers
    WHERE
      project_id = ?0
    ORDER BY
      position
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
  layers.*,
  page.total
FROM
  page
  JOIN layers ON layers.id = page.id
WHERE
  page.rank = ?2 + 1
  OR page.bytes <= ?3
ORDER BY
  page.rank;
