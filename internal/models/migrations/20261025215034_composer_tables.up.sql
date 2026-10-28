-- The languages a project can be written in.
CREATE TYPE project_language AS ENUM('en', 'fr');

CREATE TABLE projects (
  -- A UUIDv7 sorts by creation time, so the primary key also orders pages.
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  /* The user who owns the project, from the authentication service. */
  owner_id uuid NOT NULL,
  title text NOT NULL CHECK (title <> ''),
  premise text NOT NULL DEFAULT '',
  language project_language NOT NULL,
  -- Full precision lets updated_at distinguish two updates made within the same second.
  created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Lists a user's projects newest first.
CREATE INDEX projects_owner_id_idx ON projects (owner_id, id DESC);

CREATE TABLE scenes (
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
  /* The scene's place in narrative order, from 0. With the unique position key, the range caps a
  project at 500 scenes. */
  position integer NOT NULL CONSTRAINT scenes_position_check CHECK (
    position >= 0
    AND position < 500
  ),
  title text NOT NULL CHECK (title <> ''),
  /* The scene's color in the composer, as a lowercase #rrggbb hex code. */
  color text NOT NULL CHECK (color ~ '^#[0-9a-f]{6}$'),
  /* What the scene is for, written for the author. */
  description text NOT NULL DEFAULT '',
  /* Extra instructions for the model when it writes about the scene. */
  prompt text NOT NULL DEFAULT '',
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  -- A shift moves a range of positions in one statement, so uniqueness is checked at commit.
  CONSTRAINT scenes_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the cells' foreign key, which keeps a cell within its scene's project.
  CONSTRAINT scenes_project_id_id_key UNIQUE (project_id, id)
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
  CONSTRAINT layers_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the cells' foreign key, which keeps a cell within its layer's project.
  CONSTRAINT layers_project_id_id_key UNIQUE (project_id, id)
);

CREATE TABLE cells (
  project_id uuid NOT NULL,
  scene_id uuid NOT NULL,
  layer_id uuid NOT NULL,
  /* A JSON document that follows the content schema of the layer's kind. */
  content jsonb NOT NULL,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (scene_id, layer_id),
  -- Both keys carry the project, so a cell's scene and layer exist and share its project.
  CONSTRAINT cells_scene_fkey FOREIGN KEY (project_id, scene_id) REFERENCES scenes (project_id, id) ON DELETE CASCADE,
  CONSTRAINT cells_layer_fkey FOREIGN KEY (project_id, layer_id) REFERENCES layers (project_id, id) ON DELETE CASCADE
);

-- Lists a project's cells. The primary key serves the scene side; deleting a layer cascades through
-- this one.
CREATE INDEX cells_project_id_layer_id_idx ON cells (project_id, layer_id);
