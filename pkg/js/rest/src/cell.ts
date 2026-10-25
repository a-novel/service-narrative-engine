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
  sceneID: z.uuid(),
  layerID: z.uuid(),
  content: z.record(z.string(), z.unknown()),
  revision: z.int().positive(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link CellSchema} validates. */
export type Cell = z.infer<typeof CellSchema>;

/** The content of a cell, with the revision the caller read: 0 for a cell nobody wrote yet. */
export const CellWriteRequestSchema = z.object({
  sceneID: z.uuid(),
  layerID: z.uuid(),
  revision: z.int().min(0),
  content: z.union([ManuscriptContentSchema, NotesContentSchema]),
});

/** The value {@link CellWriteRequestSchema} validates. */
export type CellWriteRequest = z.infer<typeof CellWriteRequestSchema>;

/**
 * Writes a cell where a scene and a layer of the same project cross, and returns it. Throws on HTTP
 * 400 when the content breaks the schema of the layer's kind or weighs over 256 KiB, 404 when the
 * scene and the layer share no cell, and 409 when the cell moved past the revision read.
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
