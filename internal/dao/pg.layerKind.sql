-- The count runs before the filter, so it counts every layer of the project that shares the kind.
SELECT
  *
FROM
  (
    SELECT
      id,
      kind,
      count(*) OVER (
        PARTITION BY
          kind
      )::integer AS count
    FROM
      layers
    WHERE
      project_id = ?0
  ) AS kinds
WHERE
  id = ANY (ARRAY[?1]::uuid[])
LIMIT
  ?2;
