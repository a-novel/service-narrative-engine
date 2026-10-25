import type { NarrativeEngineApi } from "./api";
import { CellSchema } from "./cell";
import { LayerSchema } from "./layer";
import { SceneSchema, SceneSynopsisSchema, SceneTitleSchema } from "./scene";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** The languages a project is written in, as BCP 47 tags. */
export enum Language {
  /** English. */
  En = "en",
  /** French. */
  Fr = "fr",
}

/** Validates a {@link Language}. */
export const LanguageSchema = z.enum([Language.En, Language.Fr]);

/** One narrative, written in one language, that its owner composes scene by scene. */
export const ProjectSchema = z.object({
  id: z.uuid(),
  title: z.string(),
  premise: z.string(),
  language: LanguageSchema,
  revision: z.int().positive(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** The value {@link ProjectSchema} validates. */
export type Project = z.infer<typeof ProjectSchema>;

/** A project with every scene and layer, each list in its order, and every cell written so far. */
export const ProjectComposerSchema = ProjectSchema.extend({
  scenes: z.array(SceneSchema),
  layers: z.array(LayerSchema),
  cells: z.array(CellSchema),
});

/** The value {@link ProjectComposerSchema} validates. */
export type ProjectComposer = z.infer<typeof ProjectComposerSchema>;

/** The title of a project, which is not blank. */
export const ProjectTitleSchema = z.string().max(200).regex(/\S/);

/** What a project's narrative is about, which may be empty. */
export const ProjectPremiseSchema = z.string().max(4000);

/** The project to create, with the scenes it starts with, in order. */
export const ProjectCreateRequestSchema = z.object({
  title: ProjectTitleSchema,
  premise: ProjectPremiseSchema.optional(),
  language: LanguageSchema,
  scenes: z
    .array(z.object({ title: SceneTitleSchema.optional(), synopsis: SceneSynopsisSchema.optional() }))
    .max(100)
    .optional(),
});

/** The value {@link ProjectCreateRequestSchema} validates. */
export type ProjectCreateRequest = z.infer<typeof ProjectCreateRequestSchema>;

/** The project to read. */
export const ProjectGetRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link ProjectGetRequestSchema} validates. */
export type ProjectGetRequest = z.infer<typeof ProjectGetRequestSchema>;

/** A page of projects: at most `limit`, 20 when omitted, older than the project `after` names. */
export const ProjectListRequestSchema = z.object({
  limit: z.int().max(100).optional(),
  after: z.uuid().optional(),
});

/** The value {@link ProjectListRequestSchema} validates. */
export type ProjectListRequest = z.infer<typeof ProjectListRequestSchema>;

/** The new fields of a project, with the revision the caller read. An omitted premise is cleared. */
export const ProjectUpdateRequestSchema = z.object({
  id: z.uuid(),
  revision: z.int().positive(),
  title: ProjectTitleSchema,
  premise: ProjectPremiseSchema.optional(),
  language: LanguageSchema,
});

/** The value {@link ProjectUpdateRequestSchema} validates. */
export type ProjectUpdateRequest = z.infer<typeof ProjectUpdateRequestSchema>;

/** The project to delete, with the revision the caller read. */
export const ProjectDeleteRequestSchema = z.object({
  id: z.uuid(),
  revision: z.int().positive(),
});

/** The value {@link ProjectDeleteRequestSchema} validates. */
export type ProjectDeleteRequest = z.infer<typeof ProjectDeleteRequestSchema>;

/** Creates a project with its manuscript layer and first scenes, and returns its Scenes Composer. */
export async function projectCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectCreateRequest
): Promise<ProjectComposer> {
  return await api.fetch("/v1/projects", ProjectComposerSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/** Returns a project with its whole Scenes Composer. Throws on HTTP 404 when the caller owns no such project. */
export async function projectGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectGetRequest
): Promise<ProjectComposer> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/project?${params.toString()}`, ProjectComposerSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/** Lists the caller's projects, newest first. Pass the ID of the last project received as `after` for the next page. */
export async function projectList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectListRequest = {}
): Promise<Project[]> {
  const params = new URLSearchParams();
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.after) params.set("after", request.after);
  const query = params.toString();
  return await api.fetch(`/v1/projects${query ? `?${query}` : ""}`, z.array(ProjectSchema), {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Replaces the title, premise and language of a project. Throws on HTTP 404 for an unknown project,
 * 409 on a stale revision.
 */
export async function projectUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectUpdateRequest
): Promise<Project> {
  return await api.fetch("/v1/project", ProjectSchema, {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes a project with its scenes, layers and cells, and returns it as it was. Throws on HTTP 404
 * for an unknown project, 409 on a stale revision.
 */
export async function projectDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectDeleteRequest
): Promise<Project> {
  const params = new URLSearchParams({ id: request.id, revision: `${request.revision}` });
  return await api.fetch(`/v1/project?${params.toString()}`, ProjectSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
