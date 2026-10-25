import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** Parses an Item as returned by the service, coercing the timestamp strings to Date. */
export const ItemSchema = z.object({
  id: z.uuid(),
  name: z.string(),
  description: z.string().optional(),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
});

/** An Item resource held by the Narrative Engine service. */
export type Item = z.infer<typeof ItemSchema>;

/** Creates a new Item and returns the stored record. */
export async function itemCreate(api: NarrativeEngineApi, name: string, description?: string): Promise<Item> {
  return await api.fetch("/v0/items", ItemSchema, {
    method: "POST",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ name, description }),
  });
}

/** Fetches a single Item by ID. */
export async function itemGet(api: NarrativeEngineApi, id: string): Promise<Item> {
  const params = new URLSearchParams({ id });
  return await api.fetch(`/v0/item?${params.toString()}`, ItemSchema, { method: "GET", headers: HTTP_HEADERS.JSON });
}

/**
 * Lists Items newest first, 100 at most when limit is omitted. Pass the id of the last Item received
 * as after to read the next page.
 */
export async function itemList(api: NarrativeEngineApi, limit?: number, after?: string): Promise<Item[]> {
  const params = new URLSearchParams({ limit: `${limit || 100}` });
  if (after) params.set("after", after);
  return await api.fetch(`/v0/items?${params.toString()}`, z.array(ItemSchema), {
    method: "GET",
    headers: HTTP_HEADERS.JSON,
  });
}

/** Updates an existing Item and returns the stored record. */
export async function itemUpdate(
  api: NarrativeEngineApi,
  id: string,
  name: string,
  description?: string
): Promise<Item> {
  return await api.fetch("/v0/item", ItemSchema, {
    method: "PUT",
    headers: HTTP_HEADERS.JSON,
    body: JSON.stringify({ id, name, description }),
  });
}

/** Deletes an Item by ID and returns the record that was removed. */
export async function itemDelete(api: NarrativeEngineApi, id: string): Promise<Item> {
  const params = new URLSearchParams({ id });
  return await api.fetch(`/v0/item?${params.toString()}`, ItemSchema, { method: "DELETE", headers: HTTP_HEADERS.JSON });
}
