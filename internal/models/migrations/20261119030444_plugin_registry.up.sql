-- The registry of published plugin versions. A version is one row of plugin_versions, with one row of
-- plugin_messages per language it is translated in. Rows are insert-only: no route updates or deletes a version, so
-- a project that pins one loads the same manifest and the same modules for good.
--
-- A version is three integers and a prerelease, so each version question is one statement on the primary key: one
-- version, the latest release of each plugin, and the releases after a given one. Text would sort 1.10.0 before
-- 1.9.0.
CREATE TABLE plugin_versions (
  /* The plugin's name, which identifies it across its versions. */
  plugin text NOT NULL CHECK (plugin <> ''),
  major integer NOT NULL CHECK (major >= 0),
  minor integer NOT NULL CHECK (minor >= 0),
  patch integer NOT NULL CHECK (patch >= 0),
  /* The prerelease identifiers of a semantic version, such as dev.1. Empty for a release. */
  prerelease text NOT NULL DEFAULT '',
  /* The JSON Schema the content of an instance follows. */
  schema jsonb NOT NULL,
  /* The lists of linkable items an instance exposes, by name. */
  exports jsonb NOT NULL,
  -- A module is stored as the bytes a browser loads, so its checksum is the SHA-256 of what a read serves.
  -- PostgreSQL converts text to bytes through convert_to, which is not immutable, so a generated column cannot
  -- hash a text column.
  /* The JavaScript module that brings content from the previous version to this one. Null when the version
  needs none. */
  migration bytea CHECK (octet_length(migration) > 0),
  /* The JavaScript module that renders the plugin. */
  ui bytea NOT NULL CHECK (octet_length(ui) > 0),
  -- A stored checksum is hashed once, when the version is published. A virtual column, PostgreSQL 18's default,
  -- would hash the schema or a module on every read of its checksum.
  /* The SHA-256 of the schema as jsonb_send writes it, which changes when the schema does. A client compares it
  across versions to skip a schema it already holds. */
  schema_checksum bytea GENERATED ALWAYS AS (sha256(jsonb_send (schema))) STORED,
  /* The SHA-256 of each module, which a read serves as its ETag. */
  ui_checksum bytea GENERATED ALWAYS AS (sha256(ui)) STORED,
  migration_checksum bytea GENERATED ALWAYS AS (sha256(migration)) STORED,
  /* The bytes the version stores, with its JSON documents counted as text, so the size of the registry adds up
  without reading a module. */
  bytes bigint GENERATED ALWAYS AS (
    octet_length(schema::text) + octet_length(exports::text) + octet_length(ui) + coalesce(octet_length(migration), 0)
  ) STORED,
  published_at timestamp(0) with time zone NOT NULL,
  PRIMARY KEY (plugin, major, minor, patch, prerelease)
);

CREATE TABLE plugin_messages (
  plugin text NOT NULL,
  major integer NOT NULL,
  minor integer NOT NULL,
  patch integer NOT NULL,
  prerelease text NOT NULL,
  /* The language of the row, as a BCP 47 tag. */
  language text NOT NULL CHECK (language <> ''),
  /* The plugin's name and what it does, as the list of plugins shows them. */
  title text NOT NULL CHECK (title <> ''),
  description text NOT NULL CHECK (description <> ''),
  /* The translations of the plugin's interface and of its schema, by key. */
  messages jsonb NOT NULL,
  /* The bytes the row stores, counted as plugin_versions counts its own. */
  bytes bigint GENERATED ALWAYS AS (
    octet_length(title) + octet_length(description) + octet_length(messages::text)
  ) STORED,
  -- The key starts with the version's, so it also serves the foreign key.
  PRIMARY KEY (plugin, major, minor, patch, prerelease, language),
  CONSTRAINT plugin_messages_version_fkey FOREIGN KEY (plugin, major, minor, patch, prerelease) REFERENCES plugin_versions (plugin, major, minor, patch, prerelease) ON DELETE CASCADE
);
