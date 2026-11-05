-- A cell's summary is what a generation plans from before it reads the cell: whether its content is empty,
-- the characters of its text, its speakers, and its recap's names and the length of each version. Stored
-- generated columns compute it on write, so a plan reads neither the content nor the recap. Adding them
-- rewrites cells once under an exclusive lock.
--
-- The functions read their argument alone, so they are immutable, as a generated column requires. The size
-- counts every text, speaker and tone of the content, the members a context shows, and the speakers sort in
-- byte order, which no collation changes.
CREATE FUNCTION cell_content_size (content jsonb) RETURNS integer LANGUAGE sql IMMUTABLE RETURN (
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
);

CREATE FUNCTION cell_content_speakers (content jsonb) RETURNS text[] LANGUAGE sql IMMUTABLE RETURN ARRAY(
  SELECT DISTINCT
    speaker #>> '{}' COLLATE "C"
  FROM
    jsonb_path_query(content, '$.blocks[*].speaker') AS speaker
  ORDER BY
    1
);

ALTER TABLE cells
ADD COLUMN content_empty boolean GENERATED ALWAYS AS (
  coalesce(content ->> 'text', '') = ''
  AND NOT jsonb_path_exists(content, '$.blocks[*]')
) STORED,
ADD COLUMN content_size integer GENERATED ALWAYS AS (cell_content_size (content)) STORED,
ADD COLUMN content_speakers text[] GENERATED ALWAYS AS (cell_content_speakers (content)) STORED,
ADD COLUMN recap_names jsonb GENERATED ALWAYS AS (recap -> 'names') STORED,
ADD COLUMN recap_condensed_size integer GENERATED ALWAYS AS (char_length(recap ->> 'condensed')) STORED,
ADD COLUMN recap_compact_size integer GENERATED ALWAYS AS (char_length(recap ->> 'compact')) STORED,
ADD COLUMN recap_essentials_size integer GENERATED ALWAYS AS (char_length(recap ->> 'essentials')) STORED;
