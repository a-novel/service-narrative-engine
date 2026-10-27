import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** The kinds of layer this client knows. A kind names the content schema the cells of a layer follow. */
export enum LayerKind {
  /** The one layer of a project whose cells carry the text the manuscript exports. */
  Manuscript = "manuscript",
  /** A layer of free text the author keeps beside the manuscript. */
  Notes = "notes",
}

/**
 * A row of the Scenes Composer, at its place among the rows. Later releases add kinds, so the kind
 * stays a string and a client skips a layer whose kind is no {@link LayerKind}.
 */
export const LayerSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  position: z.int().min(0),
  kind: z.string(),
  name: z.string(),
  description: z.string(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link LayerSchema} validates. */
export type Layer = z.infer<typeof LayerSchema>;

/** The name of a layer, which is not blank. */
export const LayerNameSchema = z.string().max(100).regex(/\S/);

/** What a layer records, which may be empty. */
export const LayerDescriptionSchema = z.string().max(20000);

/** The notes layer to add, after the layer `after` names, or first without it. */
export const LayerCreateRequestSchema = z.object({
  projectID: z.uuid(),
  after: z.uuid().optional(),
  name: LayerNameSchema,
  description: LayerDescriptionSchema.optional(),
});

/** The value {@link LayerCreateRequestSchema} validates. */
export type LayerCreateRequest = z.infer<typeof LayerCreateRequestSchema>;

/** The new fields of a layer. An omitted description is cleared. */
export const LayerUpdateRequestSchema = z.object({
  id: z.uuid(),
  name: LayerNameSchema,
  description: LayerDescriptionSchema.optional(),
});

/** The value {@link LayerUpdateRequestSchema} validates. */
export type LayerUpdateRequest = z.infer<typeof LayerUpdateRequestSchema>;

/** The layer to move, after the layer `after` names, or first without it. */
export const LayerMoveRequestSchema = z.object({
  id: z.uuid(),
  after: z.uuid().optional(),
});

/** The value {@link LayerMoveRequestSchema} validates. */
export type LayerMoveRequest = z.infer<typeof LayerMoveRequestSchema>;

/** The notes layer to delete. */
export const LayerDeleteRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link LayerDeleteRequestSchema} validates. */
export type LayerDeleteRequest = z.infer<typeof LayerDeleteRequestSchema>;

/**
 * Adds a notes layer to a project and returns it at its place. Throws on HTTP 400 when `after` names
 * no layer of the project, 404 when the caller owns no such project, and 422 when the project holds
 * 20 layers already.
 */
export async function layerCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerCreateRequest
): Promise<Layer> {
  return await api.fetch("/v1/layers", LayerSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/** Replaces the name and description of a layer. Throws on HTTP 404 for an unknown layer. */
export async function layerUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerUpdateRequest
): Promise<Layer> {
  return await api.fetch("/v1/layer", LayerSchema, {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Moves a layer and returns every layer of its project in their new order. Throws on HTTP 400 when
 * `after` names the moved layer or none of the project, and 404 for an unknown layer.
 */
export async function layerMove(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerMoveRequest
): Promise<Layer[]> {
  return await api.fetch("/v1/layer/move", z.array(LayerSchema), {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes a notes layer with its cells and returns it as it was; the layers after it move up one
 * place. Throws on HTTP 404 for an unknown layer, and 422 for the manuscript layer, which stays for
 * the life of its project.
 */
export async function layerDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerDeleteRequest
): Promise<Layer> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/layer?${params.toString()}`, LayerSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
