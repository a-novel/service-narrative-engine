import type { NarrativeEngineApi } from "./api";
import { InstantSchema } from "./schema";

import { z } from "zod";

/** One end of a link, which addresses an item of the project. */
export const LinkIdentifierSchema = z.object({
  /** The instance that holds the item. The server never checks that it exists. */
  id: z.uuid(),
  /** The address of the item inside that instance, which only plugins read. */
  target: z.string().min(1),
});

/** The value {@link LinkIdentifierSchema} validates. */
export type LinkIdentifier = z.infer<typeof LinkIdentifierSchema>;

/** A relation between two items of one project. */
export const LinkSchema = z.object({
  source: LinkIdentifierSchema,
  destination: LinkIdentifierSchema,
  /** What the source is to the destination, in the words of the plugin that wrote the link. */
  relationship: z.string(),
});

/** The value {@link LinkSchema} validates. */
export type Link = z.infer<typeof LinkSchema>;

/** The links of a project, which it stores and replaces whole. */
export const ProjectLinksSchema = z.object({
  links: z.array(LinkSchema),
  /**
   * When the links last changed, as the text the server sent: null for a project that never wrote its links.
   * {@link projectLinksUpdate} sends it back as it is.
   */
  updatedAt: InstantSchema.nullable(),
});

/** The value {@link ProjectLinksSchema} validates. */
export type ProjectLinks = z.infer<typeof ProjectLinksSchema>;

/** The project whose links to read. */
export const ProjectLinksGetRequestSchema = z.object({
  projectID: z.uuid(),
});

/** The value {@link ProjectLinksGetRequestSchema} validates. */
export type ProjectLinksGetRequest = z.infer<typeof ProjectLinksGetRequestSchema>;

/** The new links of a project. An empty list clears them. */
export const ProjectLinksUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  links: z.array(LinkSchema),
  /**
   * The `updatedAt` the links carried when the client read them, as the text the server sent. Null or omitted for a
   * project that never wrote its links, which takes its first write whatever this names.
   */
  updatedAt: InstantSchema.nullable().optional(),
});

/** The value {@link ProjectLinksUpdateRequestSchema} validates. */
export type ProjectLinksUpdateRequest = z.infer<typeof ProjectLinksUpdateRequestSchema>;

/** When a write replaced the links of a project, which the next write names. */
export const ProjectLinksWrittenSchema = z.object({
  updatedAt: InstantSchema,
});

/** The value {@link ProjectLinksWrittenSchema} validates. */
export type ProjectLinksWritten = z.infer<typeof ProjectLinksWrittenSchema>;

/**
 * Returns the links of a project, and when they last changed. A project that never wrote its links returns an empty
 * list and a null `updatedAt`. Throws on HTTP 404 when the caller owns no such project.
 */
export async function projectLinksGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectLinksGetRequest
): Promise<ProjectLinks> {
  const params = new URLSearchParams({ projectID: request.projectID });
  return await api.fetch(`/v1/links?${params.toString()}`, ProjectLinksSchema, {
    method: "GET",
    accessToken,
  });
}

/**
 * Replaces the links of a project in one write, and returns their new `updatedAt`. The server checks the shape and the
 * size of the links, never that an instance or an item they name exists.
 *
 * The write applies only while the links still carry the `updatedAt` the request names. A project that never wrote
 * its links takes its first write, whatever `updatedAt` the request names. Throws on HTTP 409 once another write
 * changed the links: nothing is written, and the client reads them again. A 409 whose `tags.limit` is
 * `projectBytes` is the project's 16 MiB budget instead. Throws on 422, with the rule and the path of the link in the
 * error's `tags.invalidFields`, for links past a limit of the server or holding the NUL character, and on 404 when the
 * caller owns no such project. Throws on 429 once the caller writes documents faster than the server allows, 3
 * requests a minute on this route by default.
 */
export async function projectLinksUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectLinksUpdateRequest
): Promise<ProjectLinksWritten> {
  return await api.fetch("/v1/links", ProjectLinksWrittenSchema, {
    method: "PUT",
    accessToken,
    json: request,
  });
}
