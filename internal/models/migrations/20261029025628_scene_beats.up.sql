-- The composer's columns are beats, each split into scenes. A beat carries the color, description and
-- prompt; a scene orders within its beat and may refine the description and prompt.
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
  -- A shift moves a range of positions in one statement, so uniqueness is checked at commit.
  CONSTRAINT beats_project_id_position_key UNIQUE (project_id, position)
  DEFERRABLE INITIALLY DEFERRED,
  -- The target of the scenes' foreign key, which keeps a scene within its beat's project.
  CONSTRAINT beats_project_id_id_key UNIQUE (project_id, id)
);

-- Each scene becomes a beat of its own, under the scene's ID, which takes over its color, description
-- and prompt.
INSERT INTO
  beats (
    id,
    project_id,
    position,
    title,
    color,
    description,
    prompt,
    updated_at
  )
SELECT
  id,
  project_id,
  position,
  title,
  color,
  description,
  prompt,
  updated_at
FROM
  scenes;

-- The position key goes before the update, which moves every scene to position 0 of its beat.
ALTER TABLE scenes
DROP CONSTRAINT scenes_project_id_position_key,
DROP CONSTRAINT scenes_position_check,
ADD COLUMN beat_id uuid;

UPDATE scenes
SET
  beat_id = id,
  position = 0,
  description = '',
  prompt = '';

ALTER TABLE scenes
ALTER COLUMN beat_id
SET NOT NULL,
DROP COLUMN color,
/* The scene's place in its beat, from 0. With the unique position key, the range caps a beat at 10
scenes. */
ADD CONSTRAINT scenes_position_check CHECK (
  position >= 0
  AND position < 10
),
ADD CONSTRAINT scenes_beat_id_position_key UNIQUE (beat_id, position)
DEFERRABLE INITIALLY DEFERRED,
-- The key carries the project, so a scene's beat exists and shares its project.
ADD CONSTRAINT scenes_beat_fkey FOREIGN KEY (project_id, beat_id) REFERENCES beats (project_id, id) ON DELETE CASCADE;
