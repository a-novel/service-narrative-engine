import type { NarrativeEngineApi } from "./api";
import {
  LangSchema,
  LimitSchema,
  ModuleIDSchema,
  ModuleNamespaceSchema,
  ModuleStringSchema,
  OffsetSchema,
  SchemaSourceSchema,
  UUIDSchema,
} from "./form";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const SchemaSchema = z.object({
  id: UUIDSchema,
  projectID: UUIDSchema,
  owner: UUIDSchema.nullable(),
  module: z.string(),
  source: SchemaSourceSchema,
  data: z.record(z.string(), z.unknown()),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type Schema = z.infer<typeof SchemaSchema>;

export const SchemaVersionEntrySchema = z.object({
  id: UUIDSchema,
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type SchemaVersionEntry = z.infer<typeof SchemaVersionEntrySchema>;

export const SchemaSelectRequestSchema = z.object({
  id: UUIDSchema.optional(),
  projectID: UUIDSchema,
  module: ModuleStringSchema.optional(),
});

export type SchemaSelectRequest = z.infer<typeof SchemaSelectRequestSchema>;

export const SchemaCreateRequestSchema = z.object({
  id: UUIDSchema,
  projectID: UUIDSchema,
  module: ModuleStringSchema,
  source: SchemaSourceSchema,
  data: z.record(z.string(), z.unknown()),
});

export type SchemaCreateRequest = z.infer<typeof SchemaCreateRequestSchema>;

export const SchemaRewriteRequestSchema = z.object({
  id: UUIDSchema,
  data: z.record(z.string(), z.unknown()),
});

export type SchemaRewriteRequest = z.infer<typeof SchemaRewriteRequestSchema>;

export const SchemaListVersionsRequestSchema = z.object({
  projectID: UUIDSchema,
  moduleID: ModuleIDSchema,
  moduleNamespace: ModuleNamespaceSchema,
  limit: LimitSchema,
  offset: OffsetSchema,
});

export type SchemaListVersionsRequest = z.infer<typeof SchemaListVersionsRequestSchema>;

export const SchemaGenerateRequestSchema = z.object({
  projectID: UUIDSchema,
  module: ModuleStringSchema,
  lang: LangSchema,
});

export type SchemaGenerateRequest = z.infer<typeof SchemaGenerateRequestSchema>;

export async function schemaSelect(
  api: NarrativeEngineApi,
  accessToken: string,
  form: SchemaSelectRequest
): Promise<Schema> {
  const params = new URLSearchParams();

  params.set("projectID", form.projectID);
  if (form.id) params.set("id", form.id);
  if (form.module) params.set("module", form.module);

  return await api.fetch(`/schemas?${params.toString()}`, SchemaSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function schemaCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  form: SchemaCreateRequest
): Promise<Schema> {
  return await api.fetch("/schemas", SchemaSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}

export async function schemaRewrite(
  api: NarrativeEngineApi,
  accessToken: string,
  form: SchemaRewriteRequest
): Promise<Schema> {
  return await api.fetch("/schemas", SchemaSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PATCH",
    body: JSON.stringify(form),
  });
}

export async function schemaListVersions(
  api: NarrativeEngineApi,
  accessToken: string,
  form: SchemaListVersionsRequest
): Promise<SchemaVersionEntry[]> {
  const params = new URLSearchParams();

  params.set("projectID", form.projectID);
  params.set("moduleID", form.moduleID);
  params.set("moduleNamespace", form.moduleNamespace);
  params.set("limit", `${form.limit || 100}`);
  params.set("offset", `${form.offset || 0}`);

  return await api.fetch(`/schemas/versions?${params.toString()}`, z.array(SchemaVersionEntrySchema), {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "GET",
  });
}

export async function schemaGenerate(
  api: NarrativeEngineApi,
  accessToken: string,
  form: SchemaGenerateRequest
): Promise<Schema> {
  return await api.fetch("/schemas/generate", SchemaSchema, {
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    method: "PUT",
    body: JSON.stringify(form),
  });
}
