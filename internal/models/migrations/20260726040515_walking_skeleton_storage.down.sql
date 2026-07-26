DROP TABLE IF EXISTS manuscript_blocks;

DROP TABLE IF EXISTS manuscript_scenes;

DROP TABLE IF EXISTS manuscripts;

DROP TABLE IF EXISTS step_values;

DROP TABLE IF EXISTS generation_calls;

DROP TABLE IF EXISTS engine_versions;

DROP TABLE IF EXISTS ideas;

CREATE TABLE items (
  id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (name <> ''),
  description text,
  -- Full precision keeps created_at usable as a sort key on its own, and lets updated_at
  -- distinguish two updates made within the same second.
  created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);
