import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** A column of the Scenes Composer, at its place in narrative order. */
export const SceneSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  position: z.int().min(0),
  title: z.string(),
  synopsis: z.string(),
  revision: z.int().positive(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link SceneSchema} validates. */
export type Scene = z.infer<typeof SceneSchema>;

/** The title of a scene, which may be empty. */
export const SceneTitleSchema = z.string().max(200);

/** What happens in a scene, which may be empty. */
export const SceneSynopsisSchema = z.string().max(4000);

/** The scene to add, after the scene `after` names, or first without it. */
export const SceneCreateRequestSchema = z.object({
  projectID: z.uuid(),
  after: z.uuid().optional(),
  title: SceneTitleSchema.optional(),
  synopsis: SceneSynopsisSchema.optional(),
});

/** The value {@link SceneCreateRequestSchema} validates. */
export type SceneCreateRequest = z.infer<typeof SceneCreateRequestSchema>;

/** The new fields of a scene, with the revision the caller read. An omitted field is cleared. */
export const SceneUpdateRequestSchema = z.object({
  id: z.uuid(),
  revision: z.int().positive(),
  title: SceneTitleSchema.optional(),
  synopsis: SceneSynopsisSchema.optional(),
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

/** The scene to delete, with the revision the caller read. */
export const SceneDeleteRequestSchema = z.object({
  id: z.uuid(),
  revision: z.int().positive(),
});

/** The value {@link SceneDeleteRequestSchema} validates. */
export type SceneDeleteRequest = z.infer<typeof SceneDeleteRequestSchema>;

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

/** Replaces the title and synopsis of a scene. Throws on HTTP 404 for an unknown scene, 409 on a stale revision. */
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
 * Moves a scene and returns every scene of its project in their new order, revisions unchanged.
 * Throws on HTTP 400 when `after` names the moved scene or none of the project, and 404 for an
 * unknown scene.
 */
export async function sceneMove(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneMoveRequest
): Promise<Scene[]> {
  return await api.fetch("/v1/scene/move", z.array(SceneSchema), {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes a scene with its cells and returns it as it was; the scenes after it move up one place.
 * Throws on HTTP 404 for an unknown scene, 409 on a stale revision.
 */
export async function sceneDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneDeleteRequest
): Promise<Scene> {
  const params = new URLSearchParams({ id: request.id, revision: `${request.revision}` });
  return await api.fetch(`/v1/scene?${params.toString()}`, SceneSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
