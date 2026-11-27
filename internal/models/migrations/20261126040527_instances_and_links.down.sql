-- The instances and the links of every project are lost, and no later migration brings them back.
--
-- The default goes first: its foreign key reaches instances. Each table takes its constraints and its stored
-- bytes triggers with it.
ALTER TABLE projects
DROP COLUMN IF EXISTS default_instance_id;

DROP TABLE IF EXISTS project_links;

DROP TABLE IF EXISTS instances;

-- Dropping a table fires no trigger, so each project still counts the bytes of the rows it held. No other table
-- counted bytes before these two, so the count returns to zero.
UPDATE projects
SET
  stored_bytes = 0
WHERE
  stored_bytes <> 0;
