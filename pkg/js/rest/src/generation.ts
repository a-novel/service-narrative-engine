import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

const requiredJSONSchema = z.unknown().refine((value) => value !== undefined, {
  message: "Required",
});

/** Provider-independent lifecycle exposed by Narrative Engine. */
export const GenerationStatusSchema = z.enum(["pending", "running", "succeeded", "failed", "abandoned", "cancelled"]);

export type GenerationStatus = z.infer<typeof GenerationStatusSchema>;

/** Current owner-scoped Generation state and its opaque proposal. */
export const GenerationSchema = z
  .object({
    id: z.uuid(),
    status: GenerationStatusSchema,
    attempt: z.int().min(0),
    maxAttempts: z.int().min(1),
    proposal: requiredJSONSchema,
    failure: z.string().nullable(),
    createdAt: z.iso.datetime().transform((value) => new Date(value)),
    updatedAt: z.iso.datetime().transform((value) => new Date(value)),
    settledAt: z.iso
      .datetime()
      .transform((value) => new Date(value))
      .nullable(),
    expiresAt: z.iso
      .datetime()
      .transform((value) => new Date(value))
      .nullable(),
  })
  .strict();

export type Generation = z.infer<typeof GenerationSchema>;

/** Every client-controlled value needed to submit a Generation. */
export const GenerationSubmitRequestSchema = z
  .object({
    projectID: z.uuid(),
    idempotencyKey: z.string().refine((value) => /\S/u.test(value) && Array.from(value).length <= 256),
    instructions: z.string().refine((value) => /\S/u.test(value) && Array.from(value).length <= 32768),
    input: requiredJSONSchema,
    context: requiredJSONSchema,
    outputSchema: requiredJSONSchema,
  })
  .strict();

export type GenerationSubmitRequest = z.infer<typeof GenerationSubmitRequestSchema>;

/** Submits new work or replays the Generation identified by the idempotency key. */
export async function generationSubmit(
  api: NarrativeEngineApi,
  accessToken: string,
  request: GenerationSubmitRequest
): Promise<Generation> {
  return await api.fetch(`/v0/projects/${request.projectID}/generations`, GenerationSchema, {
    method: "POST",
    headers: {
      ...HTTP_HEADERS.JSON,
      Authorization: `Bearer ${accessToken}`,
      "Idempotency-Key": request.idempotencyKey,
    },
    body: JSON.stringify({
      instructions: request.instructions,
      input: request.input,
      context: request.context,
      outputSchema: request.outputSchema,
    }),
  });
}

/** Reads current owner-scoped Generation state. */
export async function generationGet(
  api: NarrativeEngineApi,
  accessToken: string,
  projectID: string,
  generationID: string
): Promise<Generation> {
  return await api.fetch(`/v0/projects/${projectID}/generations/${generationID}`, GenerationSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
