CREATE TABLE modules (
  -- ID of the module, as an uri-safe string.
  id text NOT NULL,
  -- Namespace to which the module belongs.
  namespace text NOT NULL,
  -- Version of the module.
  version text NOT NULL,
  -- Preversion of the module (e.g., "-beta-1", "-rc-1"). Empty string for stable versions.
  preversion text NOT NULL DEFAULT '',
  -- Description of the module.
  description text NOT NULL DEFAULT '',
  -- Schema defines the shape of the module output. It must be compatible with openAI Api structured outputs:
  -- https://platform.openai.com/docs/guides/structured-outputs
  schema json NOT NULL,
  -- UI definition to interact with the module.
  ui json NOT NULL,
  created_at timestamp(0) with time zone NOT NULL,
  -- Multiple versions of a module can be stored in any given namespace.
  PRIMARY KEY (id, namespace, version, preversion)
);
