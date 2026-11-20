import type { NarrativeEngineApi } from "./api";
import { DateSchema } from "./schema";

import { z } from "zod";

/**
 * A version of a plugin, as a semantic version without build metadata: `1.2.0`, or `1.2.0-dev.1` for a prerelease.
 * The server also refuses a number over 2147483647.
 */
export const PluginSemverSchema = z
  .string()
  .regex(
    /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?$/
  );

/** A checksum the registry computes: a SHA-256 in lowercase hex. */
const PluginChecksumSchema = z.string().regex(/^[0-9a-f]{64}$/);

/** A plugin of the registry, as its latest release describes it. */
export const PluginSchema = z.object({
  /** Identifies the plugin across its versions. */
  name: z.string(),
  /** The plugin's latest release. A prerelease is never one. */
  version: PluginSemverSchema,
  /** The plugin's title, in the language asked for, or in English when the release lacks it. */
  title: z.string(),
  /** What the plugin does, in the language of the title. */
  description: z.string(),
});

/** The value {@link PluginSchema} validates. */
export type Plugin = z.infer<typeof PluginSchema>;

/** A page of the registry's plugins, by name, with how many the registry lists in all. */
export const PluginPageSchema = z.object({
  plugins: z.array(PluginSchema),
  total: z.int(),
});

/** The value {@link PluginPageSchema} validates. */
export type PluginPage = z.infer<typeof PluginPageSchema>;

/** One version of a plugin, without its modules and its translations. */
export const PluginManifestSchema = z.object({
  plugin: z.string(),
  version: PluginSemverSchema,
  /** The JSON Schema (2020-12) the content of an instance follows. Its titles and descriptions are message keys. */
  schema: z.record(z.string(), z.unknown()),
  /** The lists of linkable items an instance exposes, by name. */
  exports: z.record(z.string(), z.unknown()),
  /** Whether the version ships a migration module, which brings content from the previous version. */
  hasMigration: z.boolean(),
  /**
   * The checksum of the schema as the server stores it. Two versions with the same one share their schema. It is not
   * the hash of `schema` as this client receives it.
   */
  schemaChecksum: PluginChecksumSchema,
  /** The checksum of the UI module's bytes. */
  uiChecksum: PluginChecksumSchema,
  publishedAt: DateSchema,
});

/** The value {@link PluginManifestSchema} validates. */
export type PluginManifest = z.infer<typeof PluginManifestSchema>;

/** The translations of one version of a plugin in one language. */
export const PluginMessagesSchema = z.object({
  /** The language of the translations, as a BCP 47 tag. */
  language: z.string(),
  title: z.string(),
  description: z.string(),
  /** The translations of the plugin's interface and of its schema, by key. */
  messages: z.record(z.string(), z.unknown()),
});

/** The value {@link PluginMessagesSchema} validates. */
export type PluginMessages = z.infer<typeof PluginMessagesSchema>;

/** One release of a plugin, as an upgrade steps through it. */
export const PluginVersionSchema = z.object({
  version: PluginSemverSchema,
  /** Whether the release ships a migration module, which an upgrade runs to reach it. */
  hasMigration: z.boolean(),
  /** The checksum of the stored schema. Two releases with the same one share their schema. */
  schemaChecksum: PluginChecksumSchema,
});

/** The value {@link PluginVersionSchema} validates. */
export type PluginVersion = z.infer<typeof PluginVersionSchema>;

/** A page of the releases of a plugin after a version, oldest first, with how many come after it in all. */
export const PluginVersionPageSchema = z.object({
  versions: z.array(PluginVersionSchema),
  total: z.int(),
});

/** The value {@link PluginVersionPageSchema} validates. */
export type PluginVersionPage = z.infer<typeof PluginVersionPageSchema>;

/**
 * A page of the registry's plugins: at most `limit`, 100 when omitted, after the first `offset`. Titles and
 * descriptions read in `language`, a BCP 47 tag, or in English when it is omitted or a plugin lacks it.
 */
export const PluginListRequestSchema = z.object({
  language: z.string().optional(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link PluginListRequestSchema} validates. */
export type PluginListRequest = z.infer<typeof PluginListRequestSchema>;

/** One version of a plugin, to read its manifest or one of its modules. */
export const PluginVersionGetRequestSchema = z.object({
  plugin: z.string().min(1),
  version: PluginSemverSchema,
});

/** The value {@link PluginVersionGetRequestSchema} validates. */
export type PluginVersionGetRequest = z.infer<typeof PluginVersionGetRequestSchema>;

/** One language of one version of a plugin. */
export const PluginMessagesGetRequestSchema = PluginVersionGetRequestSchema.extend({
  /** The language to read, as a BCP 47 tag. */
  language: z.string().min(1),
});

/** The value {@link PluginMessagesGetRequestSchema} validates. */
export type PluginMessagesGetRequest = z.infer<typeof PluginMessagesGetRequestSchema>;

/**
 * A page of the releases of a plugin that come after a version: at most `limit`, 100 when omitted, after the first
 * `offset`. The registry need not hold the `after` version.
 */
export const PluginVersionListRequestSchema = z.object({
  plugin: z.string().min(1),
  after: PluginSemverSchema,
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link PluginVersionListRequestSchema} validates. */
export type PluginVersionListRequest = z.infer<typeof PluginVersionListRequestSchema>;

/** Returns the query that names one version of a plugin. */
function versionParams(request: PluginVersionGetRequest): URLSearchParams {
  return new URLSearchParams({ plugin: request.plugin, version: request.version });
}

/**
 * Returns a page of the registry's plugins, by name, each with its latest release, and their total. A plugin that
 * only has prereleases is left out. Raise `offset` by the plugins received for the next page, while that stays
 * below the total. Throws on HTTP 422 for a language that is no BCP 47 tag.
 */
export async function pluginList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginListRequest = {}
): Promise<PluginPage> {
  const params = new URLSearchParams();
  if (request.language) params.set("language", request.language);
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  const query = params.toString();
  return await api.fetch(`/v1/plugins${query ? `?${query}` : ""}`, PluginPageSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns the manifest of one version of a plugin, a prerelease included. Throws on HTTP 404 when the registry
 * holds no such version, and 422 for a version that is no semantic version.
 */
export async function pluginManifestGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginVersionGetRequest
): Promise<PluginManifest> {
  return await api.fetch(`/v1/plugin/version?${versionParams(request).toString()}`, PluginManifestSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns the JavaScript module that renders one version of a plugin, typed `text/javascript`. A published version
 * never changes, so a caller may keep the module for good. Throws on HTTP 404 when the registry holds no such
 * version.
 */
export async function pluginUIGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginVersionGetRequest
): Promise<Blob> {
  return await api.fetchBlob(`/v1/plugin/version/ui?${versionParams(request).toString()}`, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns the JavaScript module that brings content from the previous version of a plugin to this one, typed
 * `text/javascript`. Throws on HTTP 404 when the registry holds no such version, or the version ships without a
 * migration, as its manifest's `hasMigration` tells.
 */
export async function pluginMigrationGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginVersionGetRequest
): Promise<Blob> {
  return await api.fetchBlob(`/v1/plugin/version/migration?${versionParams(request).toString()}`, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns the translations of one version of a plugin in one language, and that language alone. Throws on HTTP 404
 * when the registry holds no such version, or the version is not translated in the language. Every version is
 * translated in English.
 */
export async function pluginMessagesGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginMessagesGetRequest
): Promise<PluginMessages> {
  const params = versionParams(request);
  params.set("language", request.language);
  return await api.fetch(`/v1/plugin/version/messages?${params.toString()}`, PluginMessagesSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns a page of the releases of a plugin that come after a version, oldest first, and their total: the steps of
 * an upgrade from that version. A prerelease is never a step, and a plugin the registry lacks returns an empty page.
 * Raise `offset` by the releases received for the next page, while that stays below the total. Throws on HTTP 422
 * for an `after` that is no semantic version.
 */
export async function pluginVersionList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: PluginVersionListRequest
): Promise<PluginVersionPage> {
  const params = new URLSearchParams({ plugin: request.plugin, after: request.after });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/plugin/versions?${params.toString()}`, PluginVersionPageSchema, {
    method: "GET",
    accessToken,
  });
}
