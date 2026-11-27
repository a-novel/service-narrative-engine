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

/**
 * The name of a plugin, which identifies it across its versions: a DNS label (RFC 1035), so lowercase letters,
 * digits and hyphens, from a letter to a letter or a digit, 63 characters at most.
 */
export const PluginNameSchema = z
  .string()
  .max(63)
  .regex(/^[a-z]([-a-z0-9]*[a-z0-9])?$/);

/** A checksum the registry computes: a SHA-256 in lowercase hex. */
const PluginChecksumSchema = z.string().regex(/^[0-9a-f]{64}$/);

/**
 * The proof of who published a version of a plugin: a compact JWT of type `proof+jwt`, signed by a system API key
 * made for the purpose `publish-narrative-engine-plugin`. Its claims name the plugin and the version, and carry
 * the SHA-256 of each module in lowercase hex: `plugin`, `version`, `uiSHA256`, and `migrationSHA256` when the
 * version ships a migration.
 */
const PluginProofSchema = z.string().max(4096);

/** A plugin of the registry, as its latest release describes it. */
export const PluginSchema = z.object({
  name: PluginNameSchema,
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
  plugin: PluginNameSchema,
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
  /**
   * The proof the publisher's key signed for the version. A client verifies it before it uses the version: its
   * signature against the key `apiKeyID` names, then its claims against the version and the modules it received.
   * Absent for a version published without a key, as on a local stack.
   */
  proof: PluginProofSchema.optional(),
  /**
   * The ID of the API key that signed `proof`. A client reads the key from JSON Keys by that ID to verify the
   * proof. Absent when `proof` is.
   */
  apiKeyID: z.string().optional(),
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

/** A JSON Pointer (RFC 6901) into an exported item. */
const PluginPointerSchema = z.string().regex(/^(\/([^/~]|~[01])*)*$/);

/** One named list of linkable items an instance of a plugin exposes. */
export const PluginExportSchema = z.strictObject({
  /** The key of the version's messages that names the export. */
  name: z.string(),
  /** The path to the exported items in the content: `$`, then member names and `[*]`, a subset of JSONPath (RFC 9535). */
  select: z.string().regex(/^\$(\.[A-Za-z_][A-Za-z0-9_]*|\[\*\])*$/),
  /** The pointer to the identifier inside each item. */
  id: PluginPointerSchema,
  /** The pointer to the label inside each item. */
  label: PluginPointerSchema,
});

/** The value {@link PluginExportSchema} validates. */
export type PluginExport = z.infer<typeof PluginExportSchema>;

/**
 * The exports a version of a plugin publishes, by name. A link addresses an item as `<export>/<id>`, so a name
 * holds no slash. It mirrors the check the server runs on a published version, and both agree on the shared test
 * cases.
 */
export const PluginExportsSchema = z.record(z.string().regex(/^[A-Za-z][A-Za-z0-9_-]*$/), PluginExportSchema);

/** The value {@link PluginExportsSchema} validates. */
export type PluginExports = z.infer<typeof PluginExportsSchema>;

/** The translations of a version of a plugin in one language, as its publisher sends them. */
export const PluginVersionPublishLanguageSchema = z.strictObject({
  /** The plugin's title, as the list of plugins shows it. Not blank. */
  title: z.string().min(1).max(262144),
  /** What the plugin does, as the list of plugins shows it. Not blank. */
  description: z.string().min(1).max(262144),
  /** The translations of the plugin's interface and of its schema, by key, in 256 KiB of JSON at most. */
  messages: z.record(z.string(), z.unknown()),
});

/** The value {@link PluginVersionPublishLanguageSchema} validates. */
export type PluginVersionPublishLanguage = z.infer<typeof PluginVersionPublishLanguageSchema>;

/**
 * One whole version of a plugin, as its publisher sends it. The server also caps each JSON document at 256 KiB
 * with its numbers written in full, refuses the NUL character and what else PostgreSQL does not store, and
 * refuses a schema whose subschemas reference another document or use `$dynamicRef`, `$dynamicAnchor` or
 * `format`.
 */
export const PluginVersionPublishRequestSchema = z.strictObject({
  plugin: PluginNameSchema,
  /** The version to publish. A prerelease, such as `1.2.0-dev.1`, publishes as a release does. */
  version: PluginSemverSchema,
  /** The JSON Schema (2020-12) the content of an instance follows. */
  schema: z.record(z.string(), z.unknown()),
  exports: PluginExportsSchema,
  /** The source of the JavaScript module that renders the plugin, in 1 MiB of UTF-8 at most. */
  ui: z.string().min(1),
  /**
   * The source of the JavaScript module that brings content from the previous version to this one, in 256 KiB of
   * UTF-8 at most. Left out when the version needs none.
   */
  migration: z.string().min(1).optional(),
  /** The translations of the version by language, a BCP 47 tag each: English, as `en`, and 99 others at most. */
  languages: z
    .record(z.string(), PluginVersionPublishLanguageSchema)
    .refine((languages) => Object.hasOwn(languages, "en") && Object.keys(languages).length <= 100),
  /**
   * The proof of the version, signed by the key of the request token. Its `plugin` and `version` claims are those
   * of this request, `uiSHA256` is the SHA-256 of `ui` as UTF-8, and `migrationSHA256` that of `migration`. A
   * request with a token carries a proof, and a request without one carries none.
   */
  proof: PluginProofSchema.optional(),
});

/** The value {@link PluginVersionPublishRequestSchema} validates. */
export type PluginVersionPublishRequest = z.infer<typeof PluginVersionPublishRequestSchema>;

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
  plugin: PluginNameSchema,
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
  plugin: PluginNameSchema,
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

/**
 * Publishes one version of a plugin, whole: its manifest, its modules and its translations, or nothing. `body` is
 * the JSON of a {@link PluginVersionPublishRequest}, already encoded: the publisher's request token is signed for
 * those exact bytes, so the client sends them as they are. `requestToken` is that token, which a relay reads from
 * the request it forwards, and which a local stack whose registry takes every request does without.
 *
 * Throws on HTTP 400 for a body that holds a member the registry does not know, which stores nothing, 401 for a
 * request the publisher's key did not sign, 409 for a version the registry holds, which stays as it is, and 422 for
 * a version that breaks a rule, whose problem names the field: `proof` for a proof that does not vouch for the
 * version, or that comes without a request token. Throws on 429 once the key publishes
 * faster than the server allows, 2 versions a minute and 5 in 15 minutes by default.
 */
export async function pluginVersionPublish(
  api: NarrativeEngineApi,
  requestToken: string | undefined,
  body: BodyInit
): Promise<void> {
  await api.fetchVoid("/v1/plugin/version", {
    method: "POST",
    accessToken: requestToken,
    headers: { "Content-Type": "application/json" },
    body,
  });
}
