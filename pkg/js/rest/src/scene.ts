import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** A column of the Scenes Composer, at its place in narrative order. */
export const SceneSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  position: z.int().min(0),
  title: z.string(),
  color: z.string(),
  description: z.string(),
  prompt: z.string(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link SceneSchema} validates. */
export type Scene = z.infer<typeof SceneSchema>;

/** The title of a scene, which is not blank. */
export const SceneTitleSchema = z.string().max(200).regex(/\S/);

/** The color of a scene in the composer, as a `#rrggbb` hex code in either case. */
export const SceneColorSchema = z.string().regex(/^#[0-9A-Fa-f]{6}$/);

/** What a scene is for, written for the author. */
export const SceneDescriptionSchema = z.string().max(20000);

/** Extra instructions for the model when it writes about a scene. */
export const ScenePromptSchema = z.string().max(20000);

/** The fields of a scene: a required title and color, and an optional description and prompt. */
export const SceneFieldsSchema = z.object({
  title: SceneTitleSchema,
  color: SceneColorSchema,
  description: SceneDescriptionSchema.optional(),
  prompt: ScenePromptSchema.optional(),
});

/** The scene to add, after the scene `after` names, or first without it. */
export const SceneCreateRequestSchema = SceneFieldsSchema.extend({
  projectID: z.uuid(),
  after: z.uuid().optional(),
});

/** The value {@link SceneCreateRequestSchema} validates. */
export type SceneCreateRequest = z.infer<typeof SceneCreateRequestSchema>;

/** The new fields of a scene. An omitted description or prompt is cleared. */
export const SceneUpdateRequestSchema = SceneFieldsSchema.extend({
  id: z.uuid(),
});

/** The value {@link SceneUpdateRequestSchema} validates. */
export type SceneUpdateRequest = z.infer<typeof SceneUpdateRequestSchema>;

/** The scene to move, after the scene `after` names, or first without it. */
export const SceneMoveRequestSchema = z.object({
  id: z.uuid(),
  after: z.uuid().optional(),
});

/** The value {@link SceneMoveRequestSchema} validates. */
export type SceneMoveRequest = z.infer<typeof SceneMoveRequestSchema>;

/** The scene to delete. */
export const SceneDeleteRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link SceneDeleteRequestSchema} validates. */
export type SceneDeleteRequest = z.infer<typeof SceneDeleteRequestSchema>;

/** A page of a project's scenes, in narrative order, with how many the project holds in all. */
export const ScenePageSchema = z.object({
  scenes: z.array(SceneSchema),
  total: z.int(),
});

/** The value {@link ScenePageSchema} validates. */
export type ScenePage = z.infer<typeof ScenePageSchema>;

/** A page of a project's scenes: at most `limit`, 100 when omitted, after the first `offset`. */
export const SceneListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link SceneListRequestSchema} validates. */
export type SceneListRequest = z.infer<typeof SceneListRequestSchema>;

/**
 * Adds a scene to a project and returns it at its place. Throws on HTTP 400 when `after` names no
 * scene of the project, 404 when the caller owns no such project, and 422 when the project holds 500
 * scenes already.
 */
export async function sceneCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneCreateRequest
): Promise<Scene> {
  return await api.fetch("/v1/scenes", SceneSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/** Replaces the title, color, description and prompt of a scene. Throws on HTTP 404 for an unknown scene. */
export async function sceneUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneUpdateRequest
): Promise<Scene> {
  return await api.fetch("/v1/scene", SceneSchema, {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Moves a scene and returns it at its new position. The scenes between its old and new position shift by
 * one. Throws on HTTP 400 when `after` names the moved scene or none of the project, and 404 for an
 * unknown scene.
 */
export async function sceneMove(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneMoveRequest
): Promise<Scene> {
  return await api.fetch("/v1/scene/move", SceneSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes a scene with its cells and returns it as it was; the scenes after it move up one place.
 * Throws on HTTP 404 for an unknown scene.
 */
export async function sceneDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneDeleteRequest
): Promise<Scene> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/scene?${params.toString()}`, SceneSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Lists a page of a project's scenes, in narrative order, with their total. Raise `offset` by the page size for
 * the next page. Throws on HTTP 404 when the caller owns no such project.
 */
export async function sceneList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneListRequest
): Promise<ScenePage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/scenes?${params.toString()}`, ScenePageSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
