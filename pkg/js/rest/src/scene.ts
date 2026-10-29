import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** A subcolumn of the Scenes Composer: a part of a beat, at its place in the beat. */
export const SceneSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  beatID: z.uuid(),
  position: z.int().min(0),
  title: z.string(),
  description: z.string(),
  prompt: z.string(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link SceneSchema} validates. */
export type Scene = z.infer<typeof SceneSchema>;

/** The title of a scene, which is not blank. */
export const SceneTitleSchema = z.string().max(200).regex(/\S/);

/** What refines the beat's description for a scene, written for the author. */
export const SceneDescriptionSchema = z.string().max(20000);

/** What refines the beat's prompt for a scene, for the model. */
export const ScenePromptSchema = z.string().max(20000);

/** The fields of a scene: a required title, and an optional description and prompt that refine the beat's. */
export const SceneFieldsSchema = z.object({
  title: SceneTitleSchema,
  description: SceneDescriptionSchema.optional(),
  prompt: ScenePromptSchema.optional(),
});

/**
 * The scene to add to a beat at `position`, first when omitted. A position past the beat's last scene
 * appends it.
 */
export const SceneCreateRequestSchema = SceneFieldsSchema.extend({
  projectID: z.uuid(),
  beatID: z.uuid(),
  position: z.int().min(0).optional(),
});

/** The value {@link SceneCreateRequestSchema} validates. */
export type SceneCreateRequest = z.infer<typeof SceneCreateRequestSchema>;

/**
 * Scenes to add to a beat at `position`, in order, first when omitted. A position past the beat's last
 * scene appends them.
 */
export const ScenesCreateRequestSchema = z.object({
  projectID: z.uuid(),
  beatID: z.uuid(),
  position: z.int().min(0).optional(),
  scenes: z.array(SceneFieldsSchema).min(1).max(100),
});

/** The value {@link ScenesCreateRequestSchema} validates. */
export type ScenesCreateRequest = z.infer<typeof ScenesCreateRequestSchema>;

/** The new fields of a scene. An omitted description or prompt is cleared. */
export const SceneUpdateRequestSchema = SceneFieldsSchema.extend({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link SceneUpdateRequestSchema} validates. */
export type SceneUpdateRequest = z.infer<typeof SceneUpdateRequestSchema>;

/** Up to 100 scenes to update, each once. An omitted description or prompt is cleared. */
export const ScenesUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  scenes: z
    .array(SceneFieldsSchema.extend({ id: z.uuid() }))
    .min(1)
    .max(100),
});

/** The value {@link ScenesUpdateRequestSchema} validates. */
export type ScenesUpdateRequest = z.infer<typeof ScenesUpdateRequestSchema>;

/**
 * The scene to move, and the beat and position it moves to. The beat is its own or another of the project.
 * A position past the beat's last scene moves it last.
 */
export const SceneMoveRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  beatID: z.uuid(),
  position: z.int().min(0),
});

/** The value {@link SceneMoveRequestSchema} validates. */
export type SceneMoveRequest = z.infer<typeof SceneMoveRequestSchema>;

/** The scene to delete. */
export const SceneDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link SceneDeleteRequestSchema} validates. */
export type SceneDeleteRequest = z.infer<typeof SceneDeleteRequestSchema>;

/** Up to 100 scenes to delete, each once. */
export const ScenesDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  ids: z.array(z.uuid()).min(1).max(100),
});

/** The value {@link ScenesDeleteRequestSchema} validates. */
export type ScenesDeleteRequest = z.infer<typeof ScenesDeleteRequestSchema>;

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
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link SceneListRequestSchema} validates. */
export type SceneListRequest = z.infer<typeof SceneListRequestSchema>;

/**
 * Adds one scene to a beat and returns it at its place; the beat's scenes from there on move down one
 * place. Throws on HTTP 404 when the caller owns no such project or beat, and 422 when the beat holds 10
 * scenes already. {@link scenesCreate} adds several at once.
 */
export async function sceneCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneCreateRequest
): Promise<Scene> {
  return await api.fetch("/v1/scene", SceneSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Adds scenes to a beat and returns them in order; the beat's scenes from there on move down. They are
 * all added, or none are. Throws on HTTP 404 when the caller owns no such project or beat, and 422 when
 * the beat would hold more than 10 scenes.
 */
export async function scenesCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ScenesCreateRequest
): Promise<Scene[]> {
  return await api.fetch("/v1/scenes", z.array(SceneSchema), {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Replaces the title, description and prompt of a scene. Throws on HTTP 404 when the caller
 * owns no such project, or the project holds no such scene.
 */
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
 * Replaces the fields of several scenes at once and returns them in the order of the request. They are
 * all updated, or none are. Throws on HTTP 404 when the caller owns no such project, or the project lacks
 * one of the scenes.
 */
export async function scenesUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ScenesUpdateRequest
): Promise<Scene[]> {
  return await api.fetch("/v1/scenes", z.array(SceneSchema), {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Moves a scene to a position in a beat, its own or another, and returns it there. The scene keeps its
 * ID and its cells. Throws on HTTP 404 when the caller owns no such project, or the project holds no such
 * scene or beat, and 422 when the target beat holds 10 scenes already.
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
 * Deletes one scene with its cells and returns it as it was; the scenes after it in its beat move up one
 * place.
 * Throws on HTTP 404 when the caller owns no such project, or the project holds no such scene.
 */
export async function sceneDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: SceneDeleteRequest
): Promise<Scene> {
  const params = new URLSearchParams({ projectID: request.projectID, id: request.id });
  return await api.fetch(`/v1/scene?${params.toString()}`, SceneSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Deletes scenes with their cells and returns them as they were; the scenes after them in their beats
 * move up to close the gaps. They are all deleted, or none are. Throws on HTTP 404 when the caller owns no such
 * project, or the project lacks one of the scenes.
 */
export async function scenesDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ScenesDeleteRequest
): Promise<Scene[]> {
  const params = new URLSearchParams([["projectID", request.projectID], ...request.ids.map((id) => ["id", id])]);
  return await api.fetch(`/v1/scenes?${params.toString()}`, z.array(SceneSchema), {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Lists a page of a project's scenes, in narrative order: by beat, then by position in the beat, with their
 * total. Raise `offset` by the page size for the next page. Throws on HTTP 404 when the caller owns no such
 * project.
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
