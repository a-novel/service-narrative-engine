import type { NarrativeEngineApi } from "./api";
import { BeatRecapSourceSchema, RecapSchema } from "./recap";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/**
 * A column of the Scenes Composer, at its place in narrative order. Its scenes split it into subcolumns. A
 * beat recapped once holds its latest recap, the `updatedAt` it read in `recapBeatUpdatedAt`, and for each
 * scene the manuscript cell it read in `recapSources`.
 */
export const BeatSchema = z.object({
  id: z.uuid(),
  projectID: z.uuid(),
  position: z.int().min(0),
  title: z.string(),
  color: z.string(),
  description: z.string(),
  prompt: z.string(),
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
  recap: RecapSchema.optional(),
  recapBeatUpdatedAt: z.iso
    .datetime()
    .transform((value) => new Date(value))
    .optional(),
  recapSources: z.array(BeatRecapSourceSchema).optional(),
});

/** The value {@link BeatSchema} validates. */
export type Beat = z.infer<typeof BeatSchema>;

/** The title of a beat, which is not blank. */
export const BeatTitleSchema = z.string().max(200).regex(/\S/);

/** The color of a beat in the composer, as a `#rrggbb` hex code in either case. */
export const BeatColorSchema = z.string().regex(/^#[0-9A-Fa-f]{6}$/);

/** What a beat is for, written for the author. */
export const BeatDescriptionSchema = z.string().max(20000);

/** Extra instructions for the model when it writes about a beat. */
export const BeatPromptSchema = z.string().max(20000);

/** The fields of a beat: a required title and color, and an optional description and prompt. */
export const BeatFieldsSchema = z.object({
  title: BeatTitleSchema,
  color: BeatColorSchema,
  description: BeatDescriptionSchema.optional(),
  prompt: BeatPromptSchema.optional(),
});

/** The beat to add at `position`, first when omitted. A position past the last beat appends it. */
export const BeatCreateRequestSchema = BeatFieldsSchema.extend({
  projectID: z.uuid(),
  position: z.int().min(0).optional(),
});

/** The value {@link BeatCreateRequestSchema} validates. */
export type BeatCreateRequest = z.infer<typeof BeatCreateRequestSchema>;

/**
 * Up to 100 beats to add at `position`, in narrative order, first when omitted. A position past the last
 * beat appends them.
 */
export const BeatsCreateRequestSchema = z.object({
  projectID: z.uuid(),
  position: z.int().min(0).optional(),
  beats: z.array(BeatFieldsSchema).min(1).max(100),
});

/** The value {@link BeatsCreateRequestSchema} validates. */
export type BeatsCreateRequest = z.infer<typeof BeatsCreateRequestSchema>;

/** The new fields of a beat. An omitted description or prompt is cleared. */
export const BeatUpdateRequestSchema = BeatFieldsSchema.extend({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link BeatUpdateRequestSchema} validates. */
export type BeatUpdateRequest = z.infer<typeof BeatUpdateRequestSchema>;

/** Up to 100 beats to update, each once. An omitted description or prompt is cleared. */
export const BeatsUpdateRequestSchema = z.object({
  projectID: z.uuid(),
  beats: z
    .array(BeatFieldsSchema.extend({ id: z.uuid() }))
    .min(1)
    .max(100),
});

/** The value {@link BeatsUpdateRequestSchema} validates. */
export type BeatsUpdateRequest = z.infer<typeof BeatsUpdateRequestSchema>;

/** The beat to move, and its new position. A position past the last beat moves it last. */
export const BeatMoveRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
  position: z.int().min(0),
});

/** The value {@link BeatMoveRequestSchema} validates. */
export type BeatMoveRequest = z.infer<typeof BeatMoveRequestSchema>;

/** The beat to delete. */
export const BeatDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  id: z.uuid(),
});

/** The value {@link BeatDeleteRequestSchema} validates. */
export type BeatDeleteRequest = z.infer<typeof BeatDeleteRequestSchema>;

/** Up to 100 beats to delete, each once. */
export const BeatsDeleteRequestSchema = z.object({
  projectID: z.uuid(),
  ids: z.array(z.uuid()).min(1).max(100),
});

/** The value {@link BeatsDeleteRequestSchema} validates. */
export type BeatsDeleteRequest = z.infer<typeof BeatsDeleteRequestSchema>;

/** A page of a project's beats, in narrative order, with how many the project holds in all. */
export const BeatPageSchema = z.object({
  beats: z.array(BeatSchema),
  total: z.int(),
});

/** The value {@link BeatPageSchema} validates. */
export type BeatPage = z.infer<typeof BeatPageSchema>;

/** A page of a project's beats: at most `limit`, 100 when omitted, after the first `offset`. */
export const BeatListRequestSchema = z.object({
  projectID: z.uuid(),
  limit: z.int().min(0).max(500).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link BeatListRequestSchema} validates. */
export type BeatListRequest = z.infer<typeof BeatListRequestSchema>;

/**
 * Adds one beat to a project and returns it at its place; the beats from there on move down one place.
 * Throws on HTTP 404 when the caller owns no such project, and 422 when the project holds 200 beats
 * already. {@link beatsCreate} adds several at once.
 */
export async function beatCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatCreateRequest
): Promise<Beat> {
  return await api.fetch("/v1/beat", BeatSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Adds beats to a project and returns them in narrative order; the beats from there on move down.
 * They are all added, or none are. Throws on HTTP 404 when the caller owns no such project, and 422
 * when the project would hold more than 200 beats.
 */
export async function beatsCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatsCreateRequest
): Promise<Beat[]> {
  return await api.fetch("/v1/beats", z.array(BeatSchema), {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Replaces the title, color, description and prompt of a beat. Throws on HTTP 404 when the caller
 * owns no such project, or the project holds no such beat.
 */
export async function beatUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatUpdateRequest
): Promise<Beat> {
  return await api.fetch("/v1/beat", BeatSchema, {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Replaces the fields of several beats at once and returns them in the order of the request. They are
 * all updated, or none are. Throws on HTTP 404 when the caller owns no such project, or the project lacks
 * one of the beats.
 */
export async function beatsUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatsUpdateRequest
): Promise<Beat[]> {
  return await api.fetch("/v1/beats", z.array(BeatSchema), {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Moves a beat and returns it at its new position. The beats between its old and new position shift by
 * one. Throws on HTTP 404 when the caller owns no such project, or the project holds no such beat.
 */
export async function beatMove(api: NarrativeEngineApi, accessToken: string, request: BeatMoveRequest): Promise<Beat> {
  return await api.fetch("/v1/beat/move", BeatSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes one beat with its scenes and their cells and returns it as it was; the beats after it move up
 * one place. Throws on HTTP 404 when the caller owns no such project, or the project holds no such beat.
 */
export async function beatDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatDeleteRequest
): Promise<Beat> {
  const params = new URLSearchParams({ projectID: request.projectID, id: request.id });
  return await api.fetch(`/v1/beat?${params.toString()}`, BeatSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Deletes beats with their scenes and cells and returns them as they were; the beats after them move up to
 * close the gaps. They are all deleted, or none are. Throws on HTTP 404 when the caller owns no such
 * project, or the project lacks one of the beats.
 */
export async function beatsDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatsDeleteRequest
): Promise<Beat[]> {
  const params = new URLSearchParams([["projectID", request.projectID], ...request.ids.map((id) => ["id", id])]);
  return await api.fetch(`/v1/beats?${params.toString()}`, z.array(BeatSchema), {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Lists a page of a project's beats, in narrative order, with their total. Raise `offset` by the page size for
 * the next page. Throws on HTTP 404 when the caller owns no such project.
 */
export async function beatList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: BeatListRequest
): Promise<BeatPage> {
  const params = new URLSearchParams({ projectID: request.projectID });
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  return await api.fetch(`/v1/beats?${params.toString()}`, BeatPageSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
