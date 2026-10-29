import type { NarrativeEngineApi } from "./api";
import { ManuscriptContentSchema, NotesContentSchema } from "./content";
import { SceneDescriptionSchema, ScenePromptSchema, SceneTitleSchema } from "./scene";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** How a generation writes a target. */
export enum TargetMode {
  /** Writes the target anew, without showing the model the content it replaces. Kept blocks stay. */
  Write = "write",
  /** Starts from the target's content. */
  Revise = "revise",
}

/** Validates a {@link TargetMode}. */
export const TargetModeSchema = z.enum([TargetMode.Write, TargetMode.Revise]);

/** The fields of a scene a generation can write. */
export enum SceneField {
  /** The scene's title. */
  Title = "title",
  /** What the scene is for, written for the author. */
  Description = "description",
}

/** Validates a {@link SceneField}. */
export const SceneFieldSchema = z.enum([SceneField.Title, SceneField.Description]);

/** Where a genai generation stands. A succeeded, failed or cancelled generation has settled. */
export enum GenerationStatus {
  /** genai admitted the generation and has not started it yet. */
  Pending = "pending",
  /** The model is writing the output. */
  Running = "running",
  /** The output passed the output schema of its kind. */
  Succeeded = "succeeded",
  /** The generation failed, or its output was refused, truncated or malformed. */
  Failed = "failed",
  /** The caller cancelled the generation before it settled. */
  Cancelled = "cancelled",
}

/** Validates a {@link GenerationStatus}. */
export const GenerationStatusSchema = z.enum([
  GenerationStatus.Pending,
  GenerationStatus.Running,
  GenerationStatus.Succeeded,
  GenerationStatus.Failed,
  GenerationStatus.Cancelled,
]);

/**
 * How much of a scene a call shows, richest first. Later releases may add levels, so a scene's level stays
 * a string.
 */
export enum ContextLevel {
  /** Every cell of the scene, whole. */
  Full = "full",
  /** The manuscript as its recap's condensed version, about half of it, and the other cells whole. */
  Condensed = "condensed",
  /** The manuscript as its recap's compact version, about a tenth of it. */
  Compact = "compact",
  /** The manuscript as its recap's essentials, one sentence. */
  Essentials = "essentials",
  /** The cells of the scene other than its manuscript. */
  Outline = "outline",
}

/** A span of block positions, from start up to end excluded. A span whose ends meet inserts before start. */
export const GenerationRangeSchema = z.object({
  start: z.int().min(0),
  end: z.int().min(0),
});

/** The value {@link GenerationRangeSchema} validates. */
export type GenerationRange = z.infer<typeof GenerationRangeSchema>;

/**
 * One thing a suggest generation writes: a field of a scene, or a cell where a scene and a layer cross.
 * A manuscript target may narrow to a range of its blocks, and keep some of them unchanged.
 */
export const GenerationTargetSchema = z.object({
  sceneID: z.uuid(),
  layerID: z.uuid().optional(),
  field: SceneFieldSchema.optional(),
  mode: TargetModeSchema,
  range: GenerationRangeSchema.optional(),
  kept: z.array(z.int().min(0)).max(10000).optional(),
});

/** The value {@link GenerationTargetSchema} validates. */
export type GenerationTarget = z.infer<typeof GenerationTargetSchema>;

/** The unsaved fields of a scene, which the generation reads in place of the stored ones. */
export const SceneOverrideSchema = z.object({
  id: z.uuid(),
  title: SceneTitleSchema,
  description: SceneDescriptionSchema.optional(),
  prompt: ScenePromptSchema.optional(),
});

/** The value {@link SceneOverrideSchema} validates. */
export type SceneOverride = z.infer<typeof SceneOverrideSchema>;

/** The unsaved content of a cell, which the generation reads in place of the stored one. */
export const CellOverrideSchema = z.object({
  sceneID: z.uuid(),
  layerID: z.uuid(),
  content: z.union([ManuscriptContentSchema, NotesContentSchema]),
});

/** The value {@link CellOverrideSchema} validates. */
export type CellOverride = z.infer<typeof CellOverrideSchema>;

/** Content the author has not saved. The generation reads it, and stores none of it. */
export const GenerationOverridesSchema = z.object({
  scenes: z.array(SceneOverrideSchema).max(500).optional(),
  cells: z.array(CellOverrideSchema).max(100).optional(),
});

/** The value {@link GenerationOverridesSchema} validates. */
export type GenerationOverrides = z.infer<typeof GenerationOverridesSchema>;

/**
 * A suggest generation on a project of the caller: its targets, each once, what the author asks of it, and
 * the layers it does not read. `variant` asks for another result of the same calls.
 */
export const GenerationSubmitRequestSchema = z.object({
  projectID: z.uuid(),
  targets: z.array(GenerationTargetSchema).min(1).max(20),
  instruction: z.string().max(4000).optional(),
  exclude: z.array(z.uuid()).max(1000).optional(),
  overrides: GenerationOverridesSchema.optional(),
  variant: z.int().min(0).max(4294967295).optional(),
});

/** The value {@link GenerationSubmitRequestSchema} validates. */
export type GenerationSubmitRequest = z.infer<typeof GenerationSubmitRequestSchema>;

/**
 * One genai generation, which runs one model call. A failed generation says why, and a succeeded one carries
 * the output, which passed the output schema of its kind: parse a suggest output with `SuggestOutputSchema`.
 */
export const GenerationResultSchema = z.object({
  id: z.uuid(),
  status: GenerationStatusSchema,
  error: z.string().optional(),
  output: z.record(z.string(), z.unknown()).optional(),
});

/** The value {@link GenerationResultSchema} validates. */
export type GenerationResult = z.infer<typeof GenerationResultSchema>;

/** A block a target shows the model under a handle, at its position in the target's cell. */
export const ContextBlockSchema = z.object({
  handle: z.string(),
  position: z.int().min(0),
});

/** The value {@link ContextBlockSchema} validates. */
export type ContextBlock = z.infer<typeof ContextBlockSchema>;

/**
 * One target of a call, with the handles the model names it and its blocks by. A manuscript target always
 * has a range, the whole cell by default, and a text target none. `updatedAt` names the stored content the
 * call read, and is null for an override or a cell never written.
 */
export const ContextTargetSchema = z.object({
  handle: z.string(),
  sceneID: z.uuid(),
  layerID: z.uuid().nullable(),
  field: SceneFieldSchema.optional(),
  mode: TargetModeSchema,
  range: GenerationRangeSchema.optional(),
  kept: z.array(z.int().min(0)).optional(),
  blocks: z.array(ContextBlockSchema).optional(),
  override: z.boolean(),
  updatedAt: z.iso
    .datetime()
    .transform((value) => new Date(value))
    .nullable(),
});

/** The value {@link ContextTargetSchema} validates. */
export type ContextTarget = z.infer<typeof ContextTargetSchema>;

/** A scene a call shows, with the {@link ContextLevel} it enters at. */
export const ContextSceneSchema = z.object({
  handle: z.string(),
  sceneID: z.uuid(),
  level: z.string(),
});

/** The value {@link ContextSceneSchema} validates. */
export type ContextScene = z.infer<typeof ContextSceneSchema>;

/** A layer a call shows. */
export const ContextLayerSchema = z.object({
  handle: z.string(),
  layerID: z.uuid(),
});

/** The value {@link ContextLayerSchema} validates. */
export type ContextLayer = z.infer<typeof ContextLayerSchema>;

/**
 * What one call reads: its targets in the order of the request, the scenes it shows in narrative order, the
 * layers it shows, and the token estimate of its prompt.
 */
export const GenerationContextSchema = z.object({
  targets: z.array(ContextTargetSchema),
  scenes: z.array(ContextSceneSchema),
  layers: z.array(ContextLayerSchema),
  estimate: z.int().min(0),
});

/** The value {@link GenerationContextSchema} validates. */
export type GenerationContext = z.infer<typeof GenerationContextSchema>;

/** One model call of a generation: its genai generation, or null with the cap it waits on. */
export const GenerationCallSchema = z.object({
  generation: GenerationResultSchema.nullable(),
  waiting: z.string().optional(),
  context: GenerationContextSchema,
});

/** The value {@link GenerationCallSchema} validates. */
export type GenerationCall = z.infer<typeof GenerationCallSchema>;

/** A generation as its submit left it: its model calls, in order. */
export const GenerationSchema = z.object({
  kind: z.literal("suggest"),
  calls: z.array(GenerationCallSchema),
});

/** The value {@link GenerationSchema} validates. */
export type Generation = z.infer<typeof GenerationSchema>;

/** The call of a generation to read or cancel, by the ID of its genai generation. */
export const GenerationGetRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link GenerationGetRequestSchema} validates. */
export type GenerationGetRequest = z.infer<typeof GenerationGetRequestSchema>;

/**
 * Plans a suggest generation on a project of the caller, and submits each of its model calls to genai. Each
 * call returns with its genai generation, or with the cap it waits on, and with the handles that map its
 * output back through `suggestionsMap`. genai admits 2 calls per user at a time on the default tier: resend
 * the same request as calls settle, and genai admits the calls that wait, replays those it admitted or that
 * succeeded, and runs again those that failed or were cancelled. Throws on HTTP 400 when the request is
 * invalid, 404 when the caller owns no such project or it lacks a target's scene or layer, 422 when the
 * generation outgrows every call its kind allows, 429 when genai admitted none of its calls, with the cap
 * in the error's `tags.cap`, and 503 when genai is unavailable.
 */
export async function generationSubmit(
  api: NarrativeEngineApi,
  accessToken: string,
  request: GenerationSubmitRequest
): Promise<Generation> {
  return await api.fetch("/v1/generation", GenerationSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Reads one call of a generation through genai, until it settles. genai keeps a settled generation 6 hours.
 * Throws on HTTP 404 when the caller has no such generation, and 503 when genai is unavailable.
 */
export async function generationGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: GenerationGetRequest
): Promise<GenerationResult> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/generation?${params.toString()}`, GenerationResultSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Cancels one call of a generation through genai, so it stops costing, and returns it. A call that settled
 * first returns as it stands. Throws on HTTP 404 when the caller has no such generation, and 503 when genai is
 * unavailable.
 */
export async function generationCancel(
  api: NarrativeEngineApi,
  accessToken: string,
  request: GenerationGetRequest
): Promise<GenerationResult> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/generation?${params.toString()}`, GenerationResultSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
