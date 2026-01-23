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

export const ModuleUiSchema = z.object({
  component: z.string(),
  params: z.union([z.record(z.string(), z.unknown()), z.null()]),
  target: z.string(),
});

export type ModuleUi = z.infer<typeof ModuleUiSchema>;

export const ModuleSchema = z.object({
  id: ModuleIDSchema,
  namespace: ModuleNamespaceSchema,
  version: ModuleVersionSchema,
  preversion: ModulePreversionSchema,
  description: z.string(),
  schema: z.record(z.string(), z.unknown()),
  ui: ModuleUiSchema,
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

export const ModuleListVersionsRequestSchema = z.object({
  id: ModuleIDSchema.optional(),
  namespace: ModuleNamespaceSchema.optional(),
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

export async function moduleListVersions(
  api: NarrativeEngineApi,
  accessToken: string,
  form: ModuleListVersionsRequest
): Promise<ModuleVersionEntry[]> {
  const params = new URLSearchParams();
  params.set("limit", `${form.limit || 100}`);
  params.set("offset", `${form.offset || 0}`);

  if (form.id) params.set("id", form.id);
  if (form.namespace) params.set("namespace", form.namespace);
  if (form.version) params.set("version", form.version);
  if (form.preversion !== undefined) params.set("preversion", `${form.preversion}`);

  return await api.fetch(`/modules/versions?${params.toString()}`, z.array(ModuleVersionEntrySchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}
