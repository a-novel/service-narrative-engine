-- A summary measures the documents of a cell in the database, so a page carries none of their text. The size
-- counts every text, speaker and tone of the content, the members a context shows. Bun reads every question
-- mark as a parameter, so the paths hold no filter. The page follows the index on the project and the layer.
SELECT
  scene_id,
  layer_id,
  updated_at,
  coalesce(content ->> 'text', '') = ''
  AND NOT jsonb_path_exists(content, '$.blocks[*]') AS empty,
  (
    SELECT
      coalesce(sum(char_length(member #>> '{}')), 0)::integer
    FROM
      (
        SELECT
          jsonb_path_query(content, '$.text')
        UNION ALL
        SELECT
          jsonb_path_query(content, '$.blocks[*].text')
        UNION ALL
        SELECT
          jsonb_path_query(content, '$.blocks[*].speaker')
        UNION ALL
        SELECT
          jsonb_path_query(content, '$.blocks[*].parts[*].*')
      ) AS members (member)
  ) AS size,
  ARRAY(
    SELECT DISTINCT
      speaker #>> '{}'
    FROM
      jsonb_path_query(content, '$.blocks[*].speaker') AS speaker
    ORDER BY
      1
  ) AS speakers,
  recap -> 'names' AS recap_names,
  char_length(recap ->> 'condensed') AS recap_condensed_size,
  char_length(recap ->> 'compact') AS recap_compact_size,
  char_length(recap ->> 'essentials') AS recap_essentials_size,
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
