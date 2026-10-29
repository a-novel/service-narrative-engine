import type { NarrativeEngineApi } from "./api";
import { ManuscriptContentSchema } from "./content";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** One scene of an exported manuscript: its title and its blocks, in reading order. */
export const ManuscriptSceneSchema = ManuscriptContentSchema.extend({ title: z.string().max(200) });

/** The value {@link ManuscriptSceneSchema} validates. */
export type ManuscriptScene = z.infer<typeof ManuscriptSceneSchema>;

/**
 * The text of a narrative in manuscript format v1, which renderers turn into a novel, a screenplay or a
 * game. It mirrors the server's `manuscript.schema.json`, and both agree on the shared test cases. A
 * reader skips an appendix it does not know.
 */
export const ManuscriptSchema = z.strictObject({
  format: z.literal("manuscript"),
  formatVersion: z.literal(1),
  title: z.string().max(200),
  language: z.string(),
  scenes: z.array(ManuscriptSceneSchema).max(500),
  appendices: z.record(z.string(), z.unknown()),
});

/** The value {@link ManuscriptSchema} validates. */
export type Manuscript = z.infer<typeof ManuscriptSchema>;

/** The manuscript layer to export. A project may hold several, such as a branch of the story. */
export const ManuscriptExportRequestSchema = z.object({
  projectID: z.uuid(),
  layerID: z.uuid(),
});

/** The value {@link ManuscriptExportRequestSchema} validates. */
export type ManuscriptExportRequest = z.infer<typeof ManuscriptExportRequestSchema>;

/**
 * Exports the text one manuscript layer of a project holds: every scene in narrative order, with no blocks
 * where the layer holds no text. Throws on HTTP 404 when the caller owns no such project, or the project
 * holds no manuscript layer with that ID.
 */
export async function manuscriptExport(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ManuscriptExportRequest
): Promise<Manuscript> {
  const params = new URLSearchParams({ projectID: request.projectID, layerID: request.layerID });
  return await api.fetch(`/v1/manuscript?${params.toString()}`, ManuscriptSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
