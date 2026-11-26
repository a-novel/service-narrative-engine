-- A project is a set of plugin instances and one document of links. The server stores their JSON and bounds it,
-- and never reads what it means: the client validates, migrates and repairs it. The caps are the service's
-- configuration, so the schema holds none.
--
-- An instance pins the plugin version its content follows. No foreign key reaches the registry: the service checks
-- the format of the name and of the version, and the client resolves them.
CREATE TABLE instances (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The name of the plugin, as the registry holds it. */
  plugin text NOT NULL CHECK (plugin <> ''),
  /* The exact version of the plugin the content follows, as a semantic version. */
  version text NOT NULL CHECK (version <> ''),
  /* The display name the writer gave the instance. */
  name text NOT NULL CHECK (name <> ''),
  /* The document the plugin edits. */
  content jsonb NOT NULL,
  /* One restore point of the content, which the client writes and reads. Null until it writes one. */
  recovery jsonb,
  created_at timestamp with time zone NOT NULL,
  -- A content write names the value it read and applies only while the row still holds it, so the column keeps
  -- PostgreSQL's microseconds: at the second, two writes within one would share a value.
  /* When the version, the content or the recovery last changed. A rename leaves it as it was. */
  updated_at timestamp with time zone NOT NULL,
  /* The UTF-8 bytes the row counts in its project's stored bytes, its JSON documents counted as text. */
  bytes bigint GENERATED ALWAYS AS (
    octet_length(plugin) + octet_length(version) + octet_length(name) + octet_length(content::text) + coalesce(octet_length(recovery::text), 0)
  ) STORED,
  -- The default instance of a project reaches its row through this key, which carries the project. It also
  -- serves the list of a project's instances, oldest first.
  CONSTRAINT instances_project_id_id_key UNIQUE (project_id, id)
);

-- A project holds at most one row, written whole. A project that never wrote its links holds none.
CREATE TABLE project_links (
  project_id uuid PRIMARY KEY NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The links of the project: a JSON array of {source, destination, relationship}. */
  links jsonb NOT NULL,
  -- A write names the value it read, as an instance's content write does.
  /* When the links last changed. */
  updated_at timestamp with time zone NOT NULL,
  /* The UTF-8 bytes the row counts in its project's stored bytes, the links counted as text. */
  bytes bigint GENERATED ALWAYS AS (octet_length(links::text)) STORED
);

-- The foreign key carries the project, so a default instance is one of the project's own. It refuses the delete
-- of that instance, and a null skips it, so a project holds one default at most and none before its first
-- instance.
--
-- Deleting a project still works. Its instances go by the cascade above, within the delete's statement, and the
-- restriction then finds no project that names them: the project's row is already gone.
--
-- The new column is null in every row, so nothing is rewritten, and the table holds few rows, so validating the
-- key under its lock is brief.
ALTER TABLE projects
ADD COLUMN default_instance_id uuid,
ADD CONSTRAINT projects_default_instance_fkey FOREIGN KEY (id, default_instance_id) REFERENCES instances (project_id, id) ON DELETE RESTRICT;

-- Both tables hold a project's bytes, so each takes the three triggers that keep its stored bytes, as the
-- project_bytes_triggers migration describes them.
CREATE TRIGGER instances_stored_bytes_insert
AFTER INSERT ON instances REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER instances_stored_bytes_update
AFTER UPDATE ON instances REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER instances_stored_bytes_delete
AFTER DELETE ON instances REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER project_links_stored_bytes_insert
AFTER INSERT ON project_links REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER project_links_stored_bytes_update
AFTER UPDATE ON project_links REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER project_links_stored_bytes_delete
AFTER DELETE ON project_links REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();
