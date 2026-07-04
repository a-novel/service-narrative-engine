import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * Runtime schema for an Item returned by the narrative-engine REST API. The `createdAt`
 * and `updatedAt` timestamps arrive as ISO strings and are parsed into `Date` objects.
 */
export const ItemSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().optional(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** An Item record stored by the narrative-engine service. */
export type Item = z.infer<typeof ItemSchema>;

/** Schema for the body accepted when creating an Item. */
export const ItemCreateRequestSchema = z.object({
  name: z.string(),
  description: z.string().optional(),
});

/** Body accepted by {@link itemCreate}. */
export type ItemCreateRequest = z.infer<typeof ItemCreateRequestSchema>;

/** Schema for the parameters that select a single Item to fetch. */
export const ItemGetRequestSchema = z.object({
  id: z.uuid(),
});

/** Parameters accepted by {@link itemGet}. */
export type ItemGetRequest = z.infer<typeof ItemGetRequestSchema>;

/** Schema for the pagination parameters when listing Items. */
export const ItemListRequestSchema = z.object({
  limit: z.int().min(1).max(100).optional(),
  offset: z.int().min(0).optional(),
});

/** Parameters accepted by {@link itemList}. */
export type ItemListRequest = z.infer<typeof ItemListRequestSchema>;

/** Schema for the body accepted when updating an Item. */
export const ItemUpdateRequestSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().optional(),
});

/** Body accepted by {@link itemUpdate}. */
export type ItemUpdateRequest = z.infer<typeof ItemUpdateRequestSchema>;

/** Schema for the parameters that select a single Item to delete. */
export const ItemDeleteRequestSchema = z.object({
  id: z.uuid(),
});

/** Parameters accepted by {@link itemDelete}. */
export type ItemDeleteRequest = z.infer<typeof ItemDeleteRequestSchema>;

/** Creates an Item with the given name and optional description, returning the stored record. */
export async function itemCreate(api: NarrativeEngineApi, name: string, description?: string): Promise<Item> {
  return await api.fetch("/items", ItemSchema, {
    method: "POST",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ name, description }),
  });
}

/** Returns a single Item by its ID. */
export async function itemGet(api: NarrativeEngineApi, id: string): Promise<Item> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/item?${params.toString()}`, ItemSchema, { method: "GET", headers: HTTP_HEADERS.JSON });
}

/**
 * Returns a page of Items. `limit` defaults to 100 and `offset` to 0 when omitted; a `limit` of 0
 * also falls back to 100.
 */
export async function itemList(api: NarrativeEngineApi, limit?: number, offset?: number): Promise<Item[]> {
  const params = new URLSearchParams();
  params.set("limit", `${limit || 100}`);
  params.set("offset", `${offset || 0}`);
  return await api.fetch(`/items?${params.toString()}`, z.array(ItemSchema), {
    method: "GET",
    headers: HTTP_HEADERS.JSON,
  });
}

/** Replaces the name and description of the Item with the given ID, returning the updated record. */
export async function itemUpdate(
  api: NarrativeEngineApi,
  id: string,
  name: string,
  description?: string
): Promise<Item> {
  return await api.fetch(`/item`, ItemSchema, {
    method: "PUT",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ id, name, description }),
  });
}

/** Deletes the Item with the given ID, returning the deleted record. */
export async function itemDelete(api: NarrativeEngineApi, id: string): Promise<Item> {
  const params = new URLSearchParams();
  params.set("id", id);
  return await api.fetch(`/item?${params.toString()}`, ItemSchema, { method: "DELETE", headers: HTTP_HEADERS.JSON });
}
