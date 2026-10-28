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

/** The new content of a cell. */
export const CellWriteRequestSchema = z.object({
  projectID: z.uuid(),
  sceneID: z.uuid(),
  layerID: z.uuid(),
  content: z.union([ManuscriptContentSchema, NotesContentSchema]),
});

/** The value {@link CellWriteRequestSchema} validates. */
export type CellWriteRequest = z.infer<typeof CellWriteRequestSchema>;

/**
 * Writes a cell where a scene and a layer of the same project cross, and returns it. The first write
 * creates the cell. Throws on HTTP 400 when the content breaks the schema of the layer's kind or
 * weighs over 2 MiB, and 404 when the caller owns no such project, or
 * the project holds no such scene or layer.
 */
export async function cellWrite(
  api: NarrativeEngineApi,
  accessToken: string,
  request: CellWriteRequest
): Promise<Cell> {
  return await api.fetch("/v1/cell", CellSchema, {
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
