CREATE TABLE items (
  id uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (name <> ''),
  description text,
  created_at timestamp(0) with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  updated_at timestamp(0) with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP(0)
);
