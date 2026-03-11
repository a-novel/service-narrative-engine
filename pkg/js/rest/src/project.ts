import type { NarrativeEngineApi } from "./api";
import { LangSchema, LimitSchema, ModuleStringSchema, OffsetSchema, UUIDSchema } from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const ProjectSchema = z.object({
  id: UUIDSchema,
  owner: UUIDSchema,
  lang: LangSchema,
  title: z.string(),
  workflow: z.array(z.string()),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type Project = z.infer<typeof ProjectSchema>;

export const ProjectListRequestSchema = z.object({
  limit: LimitSchema,
  offset: OffsetSchema,
});

export type ProjectListRequest = z.infer<typeof ProjectListRequestSchema>;

export const ProjectInitRequestSchema = z.object({
  lang: LangSchema,
  title: z.string(),
  workflow: z.array(ModuleStringSchema),
});

export type ProjectInitRequest = z.infer<typeof ProjectInitRequestSchema>;

export const ProjectUpdateRequestSchema = z.object({
  id: UUIDSchema,
  title: z.string(),
  workflow: z.array(ModuleStringSchema),
});

export type ProjectUpdateRequest = z.infer<typeof ProjectUpdateRequestSchema>;

export const ProjectDeleteRequestSchema = z.object({
  id: UUIDSchema,
});

export type ProjectDeleteRequest = z.infer<typeof ProjectDeleteRequestSchema>;

export async function projectList(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ProjectListRequest
): Promise<Project[]> {
  const params = new URLSearchParams();
  params.set("limit", `${form.limit || 100}`);
  params.set("offset", `${form.offset || 0}`);

  return await api.fetch(`/projects?${params.toString()}`, z.array(ProjectSchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function projectInit(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ProjectInitRequest
): Promise<Project> {
  return await api.fetch("/projects", ProjectSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function projectUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ProjectUpdateRequest
): Promise<Project> {
  return await api.fetch("/projects", ProjectSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

export async function projectDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ProjectDeleteRequest
): Promise<Project> {
  return await api.fetch("/projects", ProjectSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "DELETE",
    body: JSON.stringify(form),
  });
}
