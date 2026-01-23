CREATE TABLE projects (
  id uuid PRIMARY KEY,
  -- User who owns the project.
  owner uuid NOT NULL,
  -- Lang of the project, ISO 639-1.
  lang varchar(5) NOT NULL,
  title text NOT NULL,
  -- Workflow is a list of module strings that define the project's workflow.
  workflow text[] NOT NULL,
  created_at timestamp(0) with time zone NOT NULL,
  updated_at timestamp(0) with time zone NOT NULL
);

-- Index on an owner for efficient listing of projects by owner
CREATE INDEX idx_projects_owner ON projects (owner, created_at DESC);
