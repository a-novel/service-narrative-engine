import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * A recap of a cell or a beat at three levels of detail, which later contexts show in place of its source.
 * It is the output of a consolidate generation, and mirrors the server's `output.consolidate.schema.json`.
 */
export const RecapSchema = z.strictObject({
  condensed: z.string().max(200000),
  compact: z.string().max(40000),
  essentials: z.string().max(1000),
  names: z.array(z.string().max(200)).max(100),
});

/** The value {@link RecapSchema} validates. */
export type Recap = z.infer<typeof RecapSchema>;

/**
 * One scene a beat's recap read, with the `updatedAt` of its manuscript cell, or null for a scene without
 * one. The stamp stays the string the server wrote, to the microsecond, since a recap write sends it back.
 */
export const BeatRecapSourceSchema = z.object({
  sceneID: z.uuid(),
  cellUpdatedAt: z.iso.datetime().nullable(),
});

/** The value {@link BeatRecapSourceSchema} validates. */
export type BeatRecapSource = z.infer<typeof BeatRecapSourceSchema>;

/**
 * The recap a consolidation wrote for a cell. `cellUpdatedAt` is the `updatedAt` of the call's target,
 * unchanged.
 */
export const CellRecapWriteRequestSchema = z.object({
  projectID: z.uuid(),
  sceneID: z.uuid(),
  layerID: z.uuid(),
  recap: RecapSchema,
  cellUpdatedAt: z.iso.datetime(),
});

/** The value {@link CellRecapWriteRequestSchema} validates. */
export type CellRecapWriteRequest = z.infer<typeof CellRecapWriteRequestSchema>;

/**
 * The recap a consolidation wrote for a beat. `beatUpdatedAt` and `sources` are the `updatedAt` and the
 * `sources` of the call's target, unchanged.
 */
export const BeatRecapWriteRequestSchema = z.object({
  projectID: z.uuid(),
  beatID: z.uuid(),
  recap: RecapSchema,
  beatUpdatedAt: z.iso.datetime(),
  sources: z.array(BeatRecapSourceSchema).max(10),
});

/** The value {@link BeatRecapWriteRequestSchema} validates. */
export type BeatRecapWriteRequest = z.infer<typeof BeatRecapWriteRequestSchema>;

/**
 * Stores the recap a succeeded consolidation wrote for a cell, stamped with the version of the cell it read,
 * and leaves the cell's `updatedAt` unchanged. The recap is fresh while the cell's `updatedAt` equals its
 * `recapCellUpdatedAt`. Throws on HTTP 400 when the recap breaks the consolidate output schema, 404 when the
 * caller owns no such project or it holds no such cell, and 409 when the recap read the cell at a version
 * older than the stored recap's, or newer than the cell's own.
 */
export async function cellRecapWrite(
  api: NarrativeEngineApi,
  accessToken: string,
  request: CellRecapWriteRequest
): Promise<void> {
  await api.fetchVoid("/v1/cell/recap", {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Stores the recap a succeeded consolidation wrote for a beat, stamped with the versions of the beat and of
 * its scenes' manuscript cells it read, and leaves the beat's `updatedAt` unchanged. Throws on HTTP 400 when
 * the recap breaks the consolidate output schema, 404 when the caller owns no such project or it holds no
 * such beat, and 409 when the recap read the beat at a version older than the stored recap's, or newer than
 * the beat's own.
 */
export async function beatRecapWrite(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatRecapWriteRequest
): Promise<void> {
  await api.fetchVoid("/v1/beat/recap", {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}
