import type { NarrativeEngineApi } from "./api";
import {
  LimitSchema,
  ModuleIDSchema,
  ModuleNamespaceSchema,
  ModulePreversionSchema,
  ModuleStringSchema,
  ModuleVersionSchema,
  OffsetSchema,
} from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const ModuleSchema = z.object({
  id: ModuleIDSchema,
  namespace: ModuleNamespaceSchema,
  version: ModuleVersionSchema,
  preversion: ModulePreversionSchema,
  description: z.string(),
  schema: z.record(z.string(), z.unknown()),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type Module = z.infer<typeof ModuleSchema>;

export const ModuleVersionEntrySchema = z.object({
  version: ModuleVersionSchema,
  preversion: ModulePreversionSchema,
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type ModuleVersionEntry = z.infer<typeof ModuleVersionEntrySchema>;

export const ModuleSelectRequestSchema = z.object({
  module: ModuleStringSchema,
});

export type ModuleSelectRequest = z.infer<typeof ModuleSelectRequestSchema>;

export const ModuleListNamespacesRequestSchema = z.object({
  limit: LimitSchema,
  offset: OffsetSchema,
});

export type ModuleListNamespacesRequest = z.infer<typeof ModuleListNamespacesRequestSchema>;

export const ModuleListIDsRequestSchema = z.object({
  namespace: ModuleNamespaceSchema,
  limit: LimitSchema,
  offset: OffsetSchema,
});

export type ModuleListIDsRequest = z.infer<typeof ModuleListIDsRequestSchema>;

export const ModuleListVersionsRequestSchema = z.object({
  id: ModuleIDSchema,
  namespace: ModuleNamespaceSchema,
  version: ModuleVersionSchema.optional(),
  preversion: z.boolean().optional(),
  limit: LimitSchema,
  offset: OffsetSchema,
});

export type ModuleListVersionsRequest = z.infer<typeof ModuleListVersionsRequestSchema>;

export async function moduleSelect(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ModuleSelectRequest
): Promise<Module> {
  const params = new URLSearchParams();
  params.set("module", form.module);

  return await api.fetch(`/modules?${params.toString()}`, ModuleSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function moduleListNamespaces(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ModuleListNamespacesRequest
): Promise<string[]> {
  const params = new URLSearchParams();
  params.set("limit", `${form.limit}`);
  params.set("offset", `${form.offset || 0}`);

  return await api.fetch(`/modules/namespaces?${params.toString()}`, z.array(z.string()), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function moduleListIDs(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ModuleListIDsRequest
): Promise<string[]> {
  const params = new URLSearchParams();
  params.set("namespace", form.namespace);
  params.set("limit", `${form.limit}`);
  params.set("offset", `${form.offset || 0}`);

  return await api.fetch(`/modules/ids?${params.toString()}`, z.array(z.string()), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function moduleListVersions(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ModuleListVersionsRequest
): Promise<ModuleVersionEntry[]> {
  const params = new URLSearchParams();
  params.set("id", form.id);
  params.set("namespace", form.namespace);
  params.set("limit", `${form.limit}`);
  params.set("offset", `${form.offset || 0}`);

  if (form.version) params.set("version", form.version);
  if (form.preversion !== undefined) params.set("preversion", `${form.preversion}`);

  return await api.fetch(`/modules/versions?${params.toString()}`, z.array(ModuleVersionEntrySchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}
