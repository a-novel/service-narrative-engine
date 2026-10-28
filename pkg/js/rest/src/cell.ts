import type { NarrativeEngineApi } from "./api";
import { ManuscriptContentSchema, NotesContentSchema } from "./content";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * What one layer says about one scene. The content follows the schema of the layer's kind: parse it
 * with {@link ManuscriptContentSchema} or {@link NotesContentSchema} once the kind is known. A kind
 * this client does not know keeps its content as a plain object.
 */
export const CellSchema = z.object({
  projectID: z.uuid(),
  sceneID: z.uuid(),
  layerID: z.uuid(),
  content: z.record(z.string(), z.unknown()),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link CellSchema} validates. */
export type Cell = z.infer<typeof CellSchema>;

/** A cell a write just stored, without the content the client already holds. */
export const WrittenCellSchema = z.object({
  sceneID: z.uuid(),
  layerID: z.uuid(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link WrittenCellSchema} validates. */
export type WrittenCell = z.infer<typeof WrittenCellSchema>;

/**
 * A page of a project's cells, by the position of their scene, then of their layer, with how many the
 * project holds in all.
 */
export const CellPageSchema = z.object({
  cells: z.array(CellSchema),
  total: z.int(),
});

/** The value {@link CellPageSchema} validates. */
export type CellPage = z.infer<typeof CellPageSchema>;

/** A page of a project's cells: at most `limit`, 100 when omitted, after the first `offset`. */
export const CellListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link CellListRequestSchema} validates. */
export type CellListRequest = z.infer<typeof CellListRequestSchema>;

/** The new content of one cell. */
export const CellWriteItemSchema = z.object({
  sceneID: z.uuid(),
  layerID: z.uuid(),
  content: z.union([ManuscriptContentSchema, NotesContentSchema]),
});

/** The value {@link CellWriteItemSchema} validates. */
export type CellWriteItem = z.infer<typeof CellWriteItemSchema>;

/** Up to 100 cells to write. A scene and a layer meet in at most one of them. */
export const CellWriteRequestSchema = z.object({
  projectID: z.uuid(),
  cells: z.array(CellWriteItemSchema).min(1).max(100),
});

/** The value {@link CellWriteRequestSchema} validates. */
export type CellWriteRequest = z.infer<typeof CellWriteRequestSchema>;

/**
 * Writes cells where scenes and layers of the project cross, and returns each written cell, without its
 * content, in the order of the request. The first write of a cell creates it, and the cells are all
 * written or none are. Throws on
 * HTTP 400 when a content breaks the schema of its layer's kind or weighs over 2 MiB, and 404 when the
 * caller owns no such project, or the project holds one of the scenes or layers not.
 */
export async function cellWrite(
  api: NarrativeEngineApi,
  accessToken: string,
  request: CellWriteRequest
): Promise<WrittenCell[]> {
  return await api.fetch("/v1/cells", z.array(WrittenCellSchema), {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Lists a page of a project's cells written so far, with their total. Raise `offset` by the page size
 * for the next page. Throws on HTTP 404 when the caller owns no such project.
 */
export async function cellList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: CellListRequest
): Promise<CellPage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/cells?${params.toString()}`, CellPageSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
