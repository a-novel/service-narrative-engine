import type { TemplateApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

export const ItemSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().optional(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

export type Item = z.infer<typeof ItemSchema>;

export const ItemCreateRequestSchema = z.object({
  name: z.string(),
  description: z.string().optional(),
});

export type ItemCreateRequest = z.infer<typeof ItemCreateRequestSchema>;

export const ItemGetRequestSchema = z.object({
  id: z.uuid(),
});

export type ItemGetRequest = z.infer<typeof ItemGetRequestSchema>;

export const ItemListRequestSchema = z.object({
  limit: z.int().min(1).max(100).optional(),
  offset: z.int().min(0).optional(),
});

export type ItemListRequest = z.infer<typeof ItemListRequestSchema>;

export const ItemUpdateRequestSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().optional(),
});

export type ItemUpdateRequest = z.infer<typeof ItemUpdateRequestSchema>;

export const ItemDeleteRequestSchema = z.object({
  id: z.uuid(),
});

export type ItemDeleteRequest = z.infer<typeof ItemDeleteRequestSchema>;

export async function itemCreate(api: TemplateApi, name: string, description?: string): Promise<Item> {
  return await api.fetch("/items", ItemSchema, {
    method: "POST",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ name, description }),
  });
}

export async function itemGet(api: TemplateApi, id: string): Promise<Item> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/item?${params.toString()}`, ItemSchema, { method: "GET", headers: HTTP_HEADERS.JSON });
}

export async function itemList(api: TemplateApi, limit?: number, offset?: number): Promise<Item[]> {
  const params = new URLSearchParams();
  params.set("limit", `${limit || 100}`);
  params.set("offset", `${offset || 0}`);
  return await api.fetch(`/items?${params.toString()}`, z.array(ItemSchema), {
    method: "GET",
    headers: HTTP_HEADERS.JSON,
  });
}

export async function itemUpdate(api: TemplateApi, id: string, name: string, description?: string): Promise<Item> {
  return await api.fetch(`/item`, ItemSchema, {
    method: "PUT",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ id, name, description }),
  });
}

export async function itemDelete(api: TemplateApi, id: string): Promise<Item> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/item?${params.toString()}`, ItemSchema, { method: "DELETE", headers: HTTP_HEADERS.JSON });
}
