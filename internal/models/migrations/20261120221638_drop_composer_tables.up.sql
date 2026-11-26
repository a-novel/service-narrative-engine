-- A project is no longer made of beats, scenes, layers and cells: the release that runs this migration stopped
-- reading and writing them. Their rows are lost, and no later migration brings them back.
--
-- A cell's foreign keys reach its scene and its layer, and a scene's reaches its beat, so the tables go in that
-- order. Each table takes its indexes, its constraints and its stored bytes triggers with it. The function those
-- triggers ran, keep_stored_bytes, stays for the tables a project holds next.
DROP TABLE cells;

DROP TABLE scenes;

DROP TABLE beats;

DROP TABLE layers;

-- Only the generated columns of cells called them.
DROP FUNCTION cell_content_speakers (jsonb);

DROP FUNCTION cell_content_size (jsonb);

-- Dropping a table fires no trigger, so each project still counts the bytes of the rows it held. The table holds
-- a few rows, so one statement resets them.
UPDATE projects
SET
  stored_bytes = 0
WHERE
  stored_bytes <> 0;
