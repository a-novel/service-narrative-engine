-- Restores the tables as the up migration found them, empty: their rows do not come back. A project then holds
-- no row, which the stored bytes the up migration reset already say.
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

CREATE TABLE layers (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The layer's place among the project's rows, from 0. With the unique position key, the range
  caps a project at 1000 layers. */
  position integer NOT NULL CONSTRAINT layers_position_check CHECK (
    position >= 0
    AND position < 1000
  ),
  /* Names the content schema the layer's cells follow, as a lowercase kebab-case label. */
  kind text NOT NULL CHECK (kind ~ '^[a-z][a-z0-9]*(-[a-z0-9]+)*$'),
  name text NOT NULL CHECK (name <> ''),
  description text NOT NULL DEFAULT '',
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bytes bigint GENERATED ALWAYS AS (octet_length(name) + octet_length(description)) STORED,
  -- A shift moves a range of positions in one statement, so uniqueness is checked at commit.
  CONSTRAINT layers_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the cells' foreign key, which keeps a cell within its layer's project.
  CONSTRAINT layers_project_id_id_key UNIQUE (project_id, id)
);

CREATE TABLE beats (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The beat's place in narrative order, from 0. With the unique position key, the range caps a
  project at 200 beats. */
  position integer NOT NULL CONSTRAINT beats_position_check CHECK (
    position >= 0
    AND position < 200
  ),
  title text NOT NULL CHECK (title <> ''),
  /* The beat's color in the composer, as a lowercase #rrggbb hex code. */
  color text NOT NULL CHECK (color ~ '^#[0-9a-f]{6}$'),
  /* What the beat is for, written for the author. */
  description text NOT NULL DEFAULT '',
  /* Extra instructions for the model when it writes about the beat. */
  prompt text NOT NULL DEFAULT '',
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  /* The beat's recap, the version of the beat it read, and the versions of the cells it read. */
  recap jsonb,
  recap_beat_updated_at timestamp with time zone,
  recap_sources jsonb,
  bytes bigint GENERATED ALWAYS AS (
    octet_length(title) + octet_length(description) + octet_length(prompt) + coalesce(octet_length(recap::text), 0) + coalesce(octet_length(recap_sources::text), 0)
  ) STORED,
  CONSTRAINT beats_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the scenes' foreign key, which keeps a scene within its beat's project.
  CONSTRAINT beats_project_id_id_key UNIQUE (project_id, id)
);

CREATE TABLE scenes (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  beat_id uuid NOT NULL,
  /* The scene's place in its beat, from 0. With the unique position key, the range caps a beat at 10
  scenes. */
  position integer NOT NULL CONSTRAINT scenes_position_check CHECK (
    position >= 0
    AND position < 10
  ),
  title text NOT NULL CHECK (title <> ''),
  description text NOT NULL DEFAULT '',
  prompt text NOT NULL DEFAULT '',
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  bytes bigint GENERATED ALWAYS AS (
    octet_length(title) + octet_length(description) + octet_length(prompt)
  ) STORED,
  CONSTRAINT scenes_beat_id_position_key UNIQUE (beat_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the cells' foreign key, which keeps a cell within its scene's project.
  CONSTRAINT scenes_project_id_id_key UNIQUE (project_id, id),
  -- The key carries the project, so a scene's beat exists and shares its project.
  CONSTRAINT scenes_beat_fkey FOREIGN KEY (project_id, beat_id) REFERENCES beats (project_id, id) ON DELETE CASCADE
);

CREATE TABLE cells (
  project_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  layer_id uuid NOT NULL,
  /* A JSON document that follows the content schema of the layer's kind. */
  content jsonb NOT NULL,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  /* The cell's recap, and the version of the cell it read. */
  recap jsonb,
  recap_cell_updated_at timestamp with time zone,
  bytes bigint GENERATED ALWAYS AS (
    octet_length(content::text) + coalesce(octet_length(recap::text), 0)
  ) STORED,
  /* What a generation planned from before it read the cell. */
  content_empty boolean GENERATED ALWAYS AS (
    coalesce(content ->> 'text', '') = ''
    AND NOT jsonb_path_exists(content, '$.blocks[*]')
  ) STORED,
  content_size integer GENERATED ALWAYS AS (cell_content_size (content)) STORED,
  content_speakers text[] GENERATED ALWAYS AS (cell_content_speakers (content)) STORED,
  recap_names jsonb GENERATED ALWAYS AS (recap -> 'names') STORED,
  recap_condensed_size integer GENERATED ALWAYS AS (char_length(recap ->> 'condensed')) STORED,
  recap_compact_size integer GENERATED ALWAYS AS (char_length(recap ->> 'compact')) STORED,
  recap_essentials_size integer GENERATED ALWAYS AS (char_length(recap ->> 'essentials')) STORED,
  PRIMARY KEY (scene_id, layer_id),
  -- Both keys carry the project, so a cell's scene and layer exist and share its project.
  CONSTRAINT cells_scene_fkey FOREIGN KEY (project_id, scene_id) REFERENCES scenes (project_id, id) ON DELETE CASCADE,
  CONSTRAINT cells_layer_fkey FOREIGN KEY (project_id, layer_id) REFERENCES layers (project_id, id) ON DELETE CASCADE
);

-- Lists a project's cells. The primary key serves the scene side; deleting a layer cascades through
-- this one.
CREATE INDEX cells_project_id_layer_id_idx ON cells (project_id, layer_id);

-- The stored bytes triggers of each table, as migration 20261117201648_project_bytes_triggers documents them.
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
