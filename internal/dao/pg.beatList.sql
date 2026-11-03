-- The page takes the beats from the offset while their bytes, summed in order, stay within the bound, and
-- always its first beat. The sums read the bytes column alone, so only the beats of the page load their
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
      beats
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
  beats.*,
  page.total
FROM
  page
  JOIN beats ON beats.id = page.id
WHERE
  page.rank = ?2 + 1
  OR page.bytes <= ?3
ORDER BY
  page.rank;
