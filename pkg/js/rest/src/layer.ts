import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** The kinds of layer this client knows. A kind names the content schema the cells of a layer follow. */
export enum LayerKind {
  /** A layer whose cells carry the text the manuscript exports. A project keeps at least one. */
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

/** A layer to add: its kind, such as {@link LayerKind.Notes}, a required name and an optional description. */
export const LayerFieldsSchema = z.object({
  kind: z.string(),
  name: LayerNameSchema,
  description: LayerDescriptionSchema.optional(),
});

/** The layer to add at `position`, first when omitted. A position past the last layer appends it. */
export const LayerCreateRequestSchema = LayerFieldsSchema.extend({
  projectID: z.uuid(),
  position: z.int().min(0).optional(),
});

/** The value {@link LayerCreateRequestSchema} validates. */
export type LayerCreateRequest = z.infer<typeof LayerCreateRequestSchema>;

/**
 * Up to 100 layers to add at `position`, in order, first when omitted. A position past the last layer
 * appends them.
 */
export const LayersCreateRequestSchema = z.object({
  projectID: z.uuid(),
  position: z.int().min(0).optional(),
  layers: z.array(LayerFieldsSchema).min(1).max(100),
});

/** The value {@link LayersCreateRequestSchema} validates. */
export type LayersCreateRequest = z.infer<typeof LayersCreateRequestSchema>;

/** The new fields of a layer. An omitted description is cleared. */
export const LayerUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  name: LayerNameSchema,
  description: LayerDescriptionSchema.optional(),
});

/** The value {@link LayerUpdateRequestSchema} validates. */
export type LayerUpdateRequest = z.infer<typeof LayerUpdateRequestSchema>;

/** Up to 100 layers to update, each once. An omitted description is cleared. */
export const LayersUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  layers: z
    .array(z.object({ id: z.uuid(), name: LayerNameSchema, description: LayerDescriptionSchema.optional() }))
    .min(1)
    .max(100),
});

/** The value {@link LayersUpdateRequestSchema} validates. */
export type LayersUpdateRequest = z.infer<typeof LayersUpdateRequestSchema>;

/** The layer to move, and its new position. A position past the last layer moves it last. */
export const LayerMoveRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  position: z.int().min(0),
});

/** The value {@link LayerMoveRequestSchema} validates. */
export type LayerMoveRequest = z.infer<typeof LayerMoveRequestSchema>;

/** The layer to delete. */
export const LayerDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link LayerDeleteRequestSchema} validates. */
export type LayerDeleteRequest = z.infer<typeof LayerDeleteRequestSchema>;

/** Up to 100 layers to delete, each once. */
export const LayersDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  ids: z.array(z.uuid()).min(1).max(100),
});

/** The value {@link LayersDeleteRequestSchema} validates. */
export type LayersDeleteRequest = z.infer<typeof LayersDeleteRequestSchema>;

/** A page of a project's layers, in order, with how many the project holds in all. */
export const LayerPageSchema = z.object({
  layers: z.array(LayerSchema),
  total: z.int(),
});

/** The value {@link LayerPageSchema} validates. */
export type LayerPage = z.infer<typeof LayerPageSchema>;

/** A page of a project's layers: at most `limit`, 100 when omitted, after the first `offset`. */
export const LayerListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link LayerListRequestSchema} validates. */
export type LayerListRequest = z.infer<typeof LayerListRequestSchema>;

/**
 * Adds one layer of any kind to a project and returns it at its place; the layers from there on move
 * down one place. Throws on HTTP 400 for a kind with no content schema, 404 when the caller owns no
 * such project, and 422 when the project holds 1000 layers already. {@link layersCreate} adds several
 * at once.
 */
export async function layerCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerCreateRequest
): Promise<Layer> {
  return await api.fetch("/v1/layer", LayerSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Adds layers of any kind to a project and returns them in order; the layers from there on move down.
 * They are all added, or none are. Throws on HTTP 400 for a kind with no content schema, 404 when the
 * caller owns no such project, and 422 when the project would hold more than 1000 layers.
 */
export async function layersCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayersCreateRequest
): Promise<Layer[]> {
  return await api.fetch("/v1/layers", z.array(LayerSchema), {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Replaces the name and description of a layer. Throws on HTTP 404 when the caller owns no such
 * project, or the project holds no such layer.
 */
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
 * Replaces the name and description of several layers at once and returns them in the order of the
 * request. They are all updated, or none are. Throws on HTTP 404 when the caller owns no such project,
 * or the project lacks one of the layers.
 */
export async function layersUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayersUpdateRequest
): Promise<Layer[]> {
  return await api.fetch("/v1/layers", z.array(LayerSchema), {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Moves a layer and returns it at its new position. The layers between its old and new position shift by
 * one. Throws on HTTP 404 when the caller owns no such project, or the project holds no such layer.
 */
export async function layerMove(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerMoveRequest
): Promise<Layer> {
  return await api.fetch("/v1/layer/move", LayerSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes one layer with its cells and returns it as it was; the layers after it move up one place.
 * Throws on HTTP 404 when the caller owns no such project, or the project holds no such layer, and 422
 * for the last layer of a required kind, such as the manuscript.
 */
export async function layerDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerDeleteRequest
): Promise<Layer> {
  const params = new URLSearchParams({ projectID: request.projectID, id: request.id });
  return await api.fetch(`/v1/layer?${params.toString()}`, LayerSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Deletes layers with their cells and returns them as they were; the layers after them move up to
 * close the gaps. They are all deleted, or none are. Throws on HTTP 404 when the caller owns no such
 * project, or the project lacks one of the layers, and 422 when the delete would leave no layer of a
 * required kind, such as the manuscript.
 */
export async function layersDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayersDeleteRequest
): Promise<Layer[]> {
  const params = new URLSearchParams([["projectID", request.projectID], ...request.ids.map((id) => ["id", id])]);
  return await api.fetch(`/v1/layers?${params.toString()}`, z.array(LayerSchema), {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Lists a page of a project's layers, in order, with their total. Raise `offset` by the page size for
 * the next page. Throws on HTTP 404 when the caller owns no such project.
 */
export async function layerList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: LayerListRequest
): Promise<LayerPage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/layers?${params.toString()}`, LayerPageSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
