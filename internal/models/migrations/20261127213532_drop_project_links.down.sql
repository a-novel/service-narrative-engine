-- The table comes back empty: the links of every project were lost with it.
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

-- The table holds a project's bytes, so it takes the three triggers that keep its stored bytes again.
CREATE TRIGGER project_links_stored_bytes_insert
AFTER INSERT ON project_links REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER project_links_stored_bytes_update
AFTER UPDATE ON project_links REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER project_links_stored_bytes_delete
AFTER DELETE ON project_links REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();
