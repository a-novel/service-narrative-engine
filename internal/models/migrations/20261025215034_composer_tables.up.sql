CREATE TABLE projects (
  -- A UUIDv7 sorts by creation time, so the primary key also orders pages.
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  /* The user who owns the project, from the authentication service. */
  owner_id uuid NOT NULL,
  title text NOT NULL CHECK (title <> ''),
  premise text NOT NULL DEFAULT '',
  /* The language the project is written in, as a BCP 47 tag. */
  language text NOT NULL CHECK (language <> ''),
  /* Raised by every write; a write names the revision it read. */
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  -- Full precision lets updated_at distinguish two updates made within the same second.
  created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Lists a user's projects newest first.
CREATE INDEX projects_owner_id_idx ON projects (owner_id, id DESC);

CREATE TABLE scenes (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The scene's place in narrative order, from 0. */
  position integer NOT NULL CHECK (position >= 0),
  title text NOT NULL DEFAULT '',
  synopsis text NOT NULL DEFAULT '',
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  -- A reorder rewrites every position in one statement, so uniqueness is checked at commit.
  CONSTRAINT scenes_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE layers (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The layer's place among the project's rows, from 0. */
  position integer NOT NULL CHECK (position >= 0),
  /* Names the content schema the layer's cells follow. */
  kind text NOT NULL CHECK (kind <> ''),
  name text NOT NULL CHECK (name <> ''),
  description text NOT NULL DEFAULT '',
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT layers_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED
);

-- The manuscript layer is the source of the export, so a project holds exactly one.
CREATE UNIQUE INDEX layers_project_id_manuscript_idx ON layers (project_id)
WHERE
  kind = 'manuscript';

CREATE TABLE cells (
  scene_id uuid NOT NULL REFERENCES scenes (id) ON DELETE CASCADE,
  layer_id uuid NOT NULL REFERENCES layers (id) ON DELETE CASCADE,
  /* A JSON document that follows the content schema of the layer's kind. */
  content jsonb NOT NULL,
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (scene_id, layer_id)
);

-- The primary key serves the scene side; deleting a layer cascades through this one.
CREATE INDEX cells_layer_id_idx ON cells (layer_id);
