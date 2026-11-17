-- A project's stored bytes sum the bytes of the beats, scenes, layers and cells it holds. Triggers keep the
-- sum, so every write counts and no DAO statement maintains it. The project's own bytes stay out of the sum,
-- in its bytes column.
--
-- Each trigger fires once per statement and reads the rows the statement changed from its transition tables.
-- Inserted rows add their bytes, deleted rows subtract theirs, and updated rows add the difference. An update
-- that changes no bytes, such as a move, leaves the project row as it is. PostgreSQL refuses transition tables
-- on a trigger with more than one event, so each table takes one trigger per event.
--
-- A delete's cascade to scenes and cells runs within the delete's statement and fires the triggers of those
-- tables. Every trigger fires before the statement returns, so a check later in the transaction reads the count
-- with the write.
CREATE FUNCTION keep_stored_bytes () RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + added.bytes
    FROM
      (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          new_rows
        GROUP BY
          project_id
      ) AS added
    WHERE
      projects.id = added.project_id;
  ELSIF TG_OP = 'DELETE' THEN
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes - removed.bytes
    FROM
      (
        SELECT
          project_id,
          sum(bytes) AS bytes
        FROM
          old_rows
        GROUP BY
          project_id
      ) AS removed
    WHERE
      projects.id = removed.project_id;
  ELSE
    UPDATE projects
    SET
      stored_bytes = projects.stored_bytes + changed.bytes
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
              new_rows
            UNION ALL
            SELECT
              project_id,
              -bytes
            FROM
              old_rows
          ) AS updated
        GROUP BY
          project_id
      ) AS changed
    WHERE
      projects.id = changed.project_id
      AND changed.bytes <> 0;
  END IF;

  RETURN NULL;
END
$$;

CREATE TRIGGER beats_stored_bytes_insert
AFTER INSERT ON beats REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER beats_stored_bytes_update
AFTER UPDATE ON beats REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER beats_stored_bytes_delete
AFTER DELETE ON beats REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER scenes_stored_bytes_insert
AFTER INSERT ON scenes REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER scenes_stored_bytes_update
AFTER UPDATE ON scenes REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER scenes_stored_bytes_delete
AFTER DELETE ON scenes REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER layers_stored_bytes_insert
AFTER INSERT ON layers REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER layers_stored_bytes_update
AFTER UPDATE ON layers REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER layers_stored_bytes_delete
AFTER DELETE ON layers REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER cells_stored_bytes_insert
AFTER INSERT ON cells REFERENCING NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER cells_stored_bytes_update
AFTER UPDATE ON cells REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();

CREATE TRIGGER cells_stored_bytes_delete
AFTER DELETE ON cells REFERENCING OLD TABLE AS old_rows FOR EACH STATEMENT
EXECUTE FUNCTION keep_stored_bytes ();
