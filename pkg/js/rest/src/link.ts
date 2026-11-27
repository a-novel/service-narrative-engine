import type { NarrativeEngineApi } from "./api";
import { InstantSchema } from "./schema";

import { z } from "zod";

/** The address of an item inside an instance, which only plugins read. */
export const LinkTargetSchema = z.string().min(1);

/** The destination of a link, which addresses an item of the project. */
export const LinkIdentifierSchema = z.object({
  /** The instance that holds the item. The server never checks that it exists. */
  id: z.uuid(),
  /** The address of the item inside that instance. */
  target: LinkTargetSchema,
});

/** The value {@link LinkIdentifierSchema} validates. */
export type LinkIdentifier = z.infer<typeof LinkIdentifierSchema>;

/** A relation from an item of the instance that stores the link to an item of an instance of the same project. */
export const LinkSchema = z.object({
  /** The item the link starts from, inside the instance that stores the link. */
  source: LinkTargetSchema,
  /** The item the link points at. Its instance may be the one that stores the link. */
  destination: LinkIdentifierSchema,
  /** What the source is to the destination, in the words of the plugin that wrote the link. */
  relationship: z.string(),
});

/** The value {@link LinkSchema} validates. */
export type Link = z.infer<typeof LinkSchema>;

/** The links an instance stores, which it replaces whole: the ones whose source is one of its items. */
export const InstanceLinksSchema = z.object({
  /** The instance that stores the links. */
  instanceID: z.uuid(),
  links: z.array(LinkSchema),
  /**
   * When the links last changed, as the text the server sent: null for an instance that never wrote its links.
   * {@link instanceLinksUpdate} sends it back as it is.
   */
  updatedAt: InstantSchema.nullable(),
});

/** The value {@link InstanceLinksSchema} validates. */
export type InstanceLinks = z.infer<typeof InstanceLinksSchema>;

/**
 * A page of the links the instances of a project store, oldest instance first, with how many instances the project
 * holds in all.
 */
export const InstanceLinksPageSchema = z.object({
  instances: z.array(InstanceLinksSchema),
  total: z.int(),
});

/** The value {@link InstanceLinksPageSchema} validates. */
export type InstanceLinksPage = z.infer<typeof InstanceLinksPageSchema>;

/**
 * A page of the links of the instances of a project: those of at most `limit` instances, 100 when omitted, after the
 * first `offset`.
 */
export const InstanceLinksListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link InstanceLinksListRequestSchema} validates. */
export type InstanceLinksListRequest = z.infer<typeof InstanceLinksListRequestSchema>;

/** The new links of an instance. An empty list clears them. */
export const InstanceLinksUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  links: z.array(LinkSchema),
  /**
   * The `updatedAt` the links carried when the client read them, as the text the server sent. Null or omitted for an
   * instance that never wrote its links: its first write names none.
   */
  updatedAt: InstantSchema.nullable().optional(),
});

/** The value {@link InstanceLinksUpdateRequestSchema} validates. */
export type InstanceLinksUpdateRequest = z.infer<typeof InstanceLinksUpdateRequestSchema>;

/** When a write replaced the links of an instance, which the next write names. */
export const InstanceLinksWrittenSchema = z.object({
  updatedAt: InstantSchema,
});

/** The value {@link InstanceLinksWrittenSchema} validates. */
export type InstanceLinksWritten = z.infer<typeof InstanceLinksWrittenSchema>;

/**
 * Returns a page of the links the instances of a project store, oldest instance first: each list with its instance,
 * and when it last changed. An instance that never wrote its links comes with an empty list and a null `updatedAt`.
 * The pages are those of `instanceList`, so one page of the default size holds every list of a project within the
 * server's default cap on instances. Throws on HTTP 404 when the caller owns no such project.
 */
export async function instanceLinksList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceLinksListRequest
): Promise<InstanceLinksPage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/links?${params.toString()}`, InstanceLinksPageSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Replaces the links an instance stores in one write, and returns their new `updatedAt`. The server checks the shape
 * and the size of the links, never that an instance or an item they name exists.
 *
 * The write applies only while the links still carry the `updatedAt` the request names: none for an instance that
 * never wrote its links. Throws on HTTP 409 once another write changed the links, or when a first write names an
 * instant: nothing is written, and the client reads the links again. A 409 whose `tags.limit` is `projectBytes` is the
 * project's 16 MiB budget instead. The links and the content of an instance carry an `updatedAt` each, so this write
 * never refuses a write of the content.
 *
 * Throws on 422, with the rule and the path of the link in the error's `tags.invalidFields`, for links past a limit of
 * the server or holding the NUL character, and on 404 when the caller owns no such project, or the project holds no
 * such instance. Throws on 429 once the caller writes documents faster than the server allows, 3 requests a minute on
 * this route by default.
 */
export async function instanceLinksUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: InstanceLinksUpdateRequest
): Promise<InstanceLinksWritten> {
  return await api.fetch("/v1/instance/links", InstanceLinksWrittenSchema, {
    method: "PUT",
    accessToken,
    json: request,
  });
}
