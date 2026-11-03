-- Each row counts the UTF-8 bytes of its text, and of its JSON documents as text, so the stored bytes of a
-- project and the bytes of a list page add up without reading the text. A stored column computes the count
-- on write; a virtual one, PostgreSQL 18's default, would read the text on every read of the count.
--
-- Adding a stored generated column rewrites its table under an exclusive lock, and the backfill updates
-- every project in one statement. The service serves no production workload yet, so both run here.
ALTER TABLE cells
ADD COLUMN bytes bigint GENERATED ALWAYS AS (
  octet_length(content::text) + coalesce(octet_length(recap::text), 0)
) STORED;

ALTER TABLE beats
ADD COLUMN bytes bigint GENERATED ALWAYS AS (
  octet_length(title) + octet_length(description) + octet_length(prompt) + coalesce(octet_length(recap::text), 0) + coalesce(octet_length(recap_sources::text), 0)
) STORED;

ALTER TABLE scenes
ADD COLUMN bytes bigint GENERATED ALWAYS AS (
  octet_length(title) + octet_length(description) + octet_length(prompt)
) STORED;

ALTER TABLE layers
ADD COLUMN bytes bigint GENERATED ALWAYS AS (octet_length(name) + octet_length(description)) STORED;

-- The stored bytes sum the bytes of every beat, scene, layer and cell the project holds, and the project's
-- own bytes stay in its row. Each write to those rows updates the sum in the statement that changes the
-- bytes, so a check later in its transaction reads the count with the write.
ALTER TABLE projects
ADD COLUMN bytes bigint GENERATED ALWAYS AS (octet_length(title) + octet_length(premise)) STORED,
ADD COLUMN stored_bytes bigint NOT NULL DEFAULT 0;

UPDATE projects
SET
  stored_bytes = held.bytes
FROM
  (
    SELECT
      project_id,
      sum(bytes) AS bytes
    FROM
      (
        SELECT
          project_id,
          bytes
        FROM
          beats
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          scenes
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          layers
        UNION ALL
        SELECT
          project_id,
          bytes
        FROM
          cells
      ) AS stored
    GROUP BY
      project_id
  ) AS held
WHERE
  projects.id = held.project_id;
