import type { NarrativeEngineApi } from "./api";
import { PluginNameSchema, PluginSemverSchema } from "./plugin";
import { DateSchema, InstantSchema, maxChars } from "./schema";

import { z } from "zod";

/** The most characters the display name of an instance holds. */
export const INSTANCE_NAME_CHARS_MAX = 80;

/**
 * The display name of an instance. It is empty when the writer gave none, and the client then shows a name of its
 * own. A name of only whitespace is refused.
 */
export const InstanceNameSchema = maxChars(INSTANCE_NAME_CHARS_MAX).regex(/^$|\S/);

/** The most characters the version of an instance holds: the longest version node-semver reads. */
export const INSTANCE_VERSION_CHARS_MAX = 256;

/** The exact version of the plugin the content of an instance follows. */
export const InstanceVersionSchema = PluginSemverSchema.max(INSTANCE_VERSION_CHARS_MAX);

/**
 * The rules a document breaks when the server refuses it, as a 422 error's `tags.invalidFields` names them beside the
 * field: `content`, `links`, or a path such as `instances[2].content` or `links[3].relationship`. The limits are the
 * server's configuration, so the client holds no copy of them.
 */
export enum DocumentRefusal {
  /**
   * The document, or the text, is over its size, in bytes. A document counts each number as the server stores it,
   * written in full: an exponent adds as many bytes.
   */
  MaxBytes = "maxbytes",
  /** The list holds too many items, or the text too many characters. */
  Max = "max",
  /** The document nests its arrays and objects too deep. */
  MaxDepth = "maxdepth",
  /** The document is not JSON in UTF-8. */
  JSON = "json",
  /** A string of the document holds the NUL character, which the server stores in no document. */
  NoNul = "nonul",
  /**
   * A string of the document holds half of a surrogate pair, as a string cut inside an emoji does. The server
   * stores none: `String.prototype.toWellFormed` replaces them.
   */
  NoLoneSurrogate = "nolonesurrogate",
  /** A number of the document holds more than 16383 digits, its exponent written out. No JavaScript number does. */
  MaxDigits = "maxdigits",
}

/** What a project knows of an instance without reading its content. */
export const InstanceDescriptorSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  /** The name of the plugin, as the registry holds it. */
  plugin: PluginNameSchema,
  version: InstanceVersionSchema,
  /** The display name the writer gave the instance. */
  name: z.string(),
  createdAt: DateSchema,
  /**
   * When the version or the content last changed, as the text the server sent. A rename leaves it as it was.
   * {@link instanceContentUpdate} sends it back as it is.
   */
  updatedAt: InstantSchema,
});

/** The value {@link InstanceDescriptorSchema} validates. */
export type InstanceDescriptor = z.infer<typeof InstanceDescriptorSchema>;

/** A page of the instances of a project, oldest first, with how many the project holds in all. */
export const InstancePageSchema = z.object({
  instances: z.array(InstanceDescriptorSchema),
  total: z.int(),
});

/** The value {@link InstancePageSchema} validates. */
export type InstancePage = z.infer<typeof InstancePageSchema>;

/**
 * One use of a plugin in a project, with its content. The server stores the content without reading what it means,
 * and returns it as a JSON value whose spacing, key order and number spelling may differ from what was written:
 * validate `content` against the schema of its plugin version.
 */
export const InstanceSchema = InstanceDescriptorSchema.extend({
  /** The document the plugin edits. */
  content: z.unknown(),
});

/** The value {@link InstanceSchema} validates. */
export type Instance = z.infer<typeof InstanceSchema>;

/** One instance to create, the project aside. */
export const InstancesCreateItemSchema = z.object({
  plugin: PluginNameSchema,
  version: InstanceVersionSchema,
  /** Omitted, the instance has no name. */
  name: InstanceNameSchema.optional(),
  /** The document the plugin edits: any JSON value. */
  content: z.unknown(),
});

/** The value {@link InstancesCreateItemSchema} validates. */
export type InstancesCreateItem = z.infer<typeof InstancesCreateItemSchema>;

/** The instance to create. */
export const InstanceCreateRequestSchema = InstancesCreateItemSchema.extend({
  projectID: z.uuid(),
});

/** The value {@link InstanceCreateRequestSchema} validates. */
export type InstanceCreateRequest = z.infer<typeof InstanceCreateRequestSchema>;

/** The instances to create in a project, in order. */
export const InstancesCreateRequestSchema = z.object({
  projectID: z.uuid(),
  instances: z.array(InstancesCreateItemSchema).min(1),
});

/** The value {@link InstancesCreateRequestSchema} validates. */
export type InstancesCreateRequest = z.infer<typeof InstancesCreateRequestSchema>;

/** A page of the instances of a project: at most `limit`, 100 when omitted, after the first `offset`. */
export const InstanceListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link InstanceListRequestSchema} validates. */
export type InstanceListRequest = z.infer<typeof InstanceListRequestSchema>;

/** The instance to read. */
export const InstanceGetRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link InstanceGetRequestSchema} validates. */
export type InstanceGetRequest = z.infer<typeof InstanceGetRequestSchema>;

/** The new name of an instance. Omitted or empty, the instance has none. */
export const InstanceUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  name: InstanceNameSchema.optional(),
});

/** The value {@link InstanceUpdateRequestSchema} validates. */
export type InstanceUpdateRequest = z.infer<typeof InstanceUpdateRequestSchema>;

/** The new version and content of an instance. */
export const InstanceContentUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  /** The version a migration brought the content to. */
  version: InstanceVersionSchema,
  /** The document the plugin edits: any JSON value. */
  content: z.unknown(),
  /** The `updatedAt` the instance carried when the client read it, as the text the server sent. */
  updatedAt: InstantSchema,
});

/** The value {@link InstanceContentUpdateRequestSchema} validates. */
export type InstanceContentUpdateRequest = z.infer<typeof InstanceContentUpdateRequestSchema>;

/** The instance to delete. */
export const InstanceDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link InstanceDeleteRequestSchema} validates. */
export type InstanceDeleteRequest = z.infer<typeof InstanceDeleteRequestSchema>;

/**
 * Creates an instance in a project, and returns it without its content. The first instance of a project becomes its
 * default. The server checks the format of the plugin and of the version, never that the registry holds them.
 *
 * Throws on HTTP 404 when the caller owns no such project, on 422 with a {@link DocumentRefusal} under `content` in
 * the error's `tags.invalidFields` for a content it refuses, and on 409 with `instances` or `projectBytes` in the
 * error's `tags.limit` when the project holds as many instances as it may, or would pass its 16 MiB budget of stored
 * bytes. Throws on 429 once the caller writes documents faster than the server allows, 3 requests a minute on this
 * route by default.
 */
export async function instanceCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceCreateRequest
): Promise<InstanceDescriptor> {
  return await api.fetch("/v1/instance", InstanceDescriptorSchema, {
    method: "POST",
    accessToken,
    json: request,
  });
}

/**
 * Creates instances in a project, all of them or none, and returns them in the order of the request, without their
 * content. A client seeds a new project with it. It throws as {@link instanceCreate} does, and names a refused content
 * by its place in the request, as `instances[2].content`.
 */
export async function instancesCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstancesCreateRequest
): Promise<InstanceDescriptor[]> {
  return await api.fetch("/v1/instances", z.array(InstanceDescriptorSchema), {
    method: "POST",
    accessToken,
    json: request,
  });
}

/**
 * Returns a page of the instances of a project, oldest first, without their content, and their total. Raise `offset`
 * by the instances received for the next page, while that stays below the total. Throws on HTTP 404 when the caller
 * owns no such project.
 */
export async function instanceList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceListRequest
): Promise<InstancePage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/instances?${params.toString()}`, InstancePageSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Returns an instance, with its content. Throws on HTTP 404 when the caller owns no such project, or the project
 * holds no such instance.
 */
export async function instanceGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceGetRequest
): Promise<Instance> {
  const params = new URLSearchParams({ projectID: request.projectID, id: request.id });
  return await api.fetch(`/v1/instance?${params.toString()}`, InstanceSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Renames an instance, or clears its name, and returns it without its content. A rename leaves the instance's
 * `updatedAt`, so it refuses no write of the content. Throws on HTTP 404 when the caller owns no such project, or the project holds no such
 * instance.
 */
export async function instanceUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceUpdateRequest
): Promise<InstanceDescriptor> {
  return await api.fetch("/v1/instance", InstanceDescriptorSchema, {
    method: "PUT",
    accessToken,
    json: request,
  });
}

/**
 * Replaces the version and the content of an instance in one write, and returns the instance without its content.
 * Its `updatedAt` is the one the next write names.
 *
 * The write applies only while the instance still carries the `updatedAt` the request names. Throws on HTTP 409 once
 * another write changed the instance: nothing is written, and the client reads the instance again. A 409 whose
 * `tags.limit` is `projectBytes` is the project's 16 MiB budget instead. Throws on 422 with a {@link DocumentRefusal}
 * under `content` in the error's `tags.invalidFields` for a content the server refuses, and on 404 when the caller
 * owns no such project, or the project holds no such instance.
 *
 * Throws on 429 once the caller writes documents faster than the server allows, 3 requests a minute on this route by
 * default: a client keeps the writer's edits itself and saves them at intervals.
 */
export async function instanceContentUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceContentUpdateRequest
): Promise<InstanceDescriptor> {
  return await api.fetch("/v1/instance/content", InstanceDescriptorSchema, {
    method: "PUT",
    accessToken,
    json: request,
  });
}

/**
 * Deletes an instance and its content, and returns it without its content. The links that name it stay: the client prunes
 * them. Throws on HTTP 409 for the default instance of the project, which stays until `projectDefaultUpdate` chooses
 * another, and on 404 when the caller owns no such project, or the project holds no such instance.
 */
export async function instanceDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceDeleteRequest
): Promise<InstanceDescriptor> {
  const params = new URLSearchParams({ projectID: request.projectID, id: request.id });
  return await api.fetch(`/v1/instance?${params.toString()}`, InstanceDescriptorSchema, {
    method: "DELETE",
    accessToken,
  });
}
