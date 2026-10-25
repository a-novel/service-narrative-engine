CREATE TABLE items (
  -- A UUIDv7 sorts by creation time, so the primary key also orders pages.
  id uuid PRIMARY KEY NOT NULL DEFAULT uuidv7(),
  name text NOT NULL CHECK (name <> ''),
  description text,
  -- Full precision lets updated_at distinguish two updates made within the same second.
  created_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);
