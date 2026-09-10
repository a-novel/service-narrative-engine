import type { NarrativeEngineApi } from "./api";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

const requiredJSONSchema = z.unknown().refine((value) => value !== undefined, {
  message: "Required",
});

function boundedUnicodeString(maxLength: number) {
  return z
    .string()
    .refine((value) => /\S/u.test(value), { message: "Must contain a non-whitespace character" })
    .refine((value) => Array.from(value).length <= maxLength, {
      message: `Must contain at most ${maxLength} Unicode characters`,
    });
}

/** The complete, stable Idea value accepted by every Project. */
export const IdeaValueSchema = z
  .object({
    title: boundedUnicodeString(128),
    genre: boundedUnicodeString(128),
    seed: boundedUnicodeString(32768),
  })
  .strict();

export type IdeaValue = z.infer<typeof IdeaValueSchema>;

/** One rich-text presentation range inside a Manuscript text block. */
export const ManuscriptTextMarkSchema = z
  .object({
    type: z.enum(["bold", "italic", "underline", "strikethrough"]),
    start: z.int().min(0).max(32768),
    end: z.int().min(1).max(32768),
  })
  .strict();

export type ManuscriptTextMark = z.infer<typeof ManuscriptTextMarkSchema>;

/** The initial Manuscript block contract: rich text plus free-form editor metadata. */
export const ManuscriptTextBlockSchema = z
  .object({
    type: z.literal("text"),
    metadata: z.record(z.string(), z.unknown()),
    data: z
      .object({
        text: boundedUnicodeString(32768),
        marks: z.array(ManuscriptTextMarkSchema),
      })
      .strict(),
  })
  .strict()
  .superRefine((block, context) => {
    const textLength = Array.from(block.data.text).length;

    block.data.marks.forEach((mark, index) => {
      if (mark.start >= mark.end || mark.end > textLength) {
        context.addIssue({
          code: "custom",
          message: "Mark must be a non-empty range inside the block text",
          path: ["data", "marks", index],
        });
      }
    });
  });

export type ManuscriptTextBlock = z.infer<typeof ManuscriptTextBlockSchema>;

/** The complete, stable Manuscript document accepted by every Project. */
export const ManuscriptValueSchema = z
  .object({
    blocks: z.array(ManuscriptTextBlockSchema).min(1),
  })
  .strict();

export type ManuscriptValue = z.infer<typeof ManuscriptValueSchema>;

/** One immutable Idea save. */
export const IdeaVersionSchema = z
  .object({
    id: z.uuid(),
    projectID: z.uuid(),
    value: IdeaValueSchema,
    createdAt: z.iso.datetime().transform((value) => new Date(value)),
  })
  .strict();

export type IdeaVersion = z.infer<typeof IdeaVersionSchema>;

/** One immutable opaque Step save. */
export const StepValueVersionSchema = z
  .object({
    id: z.uuid(),
    projectID: z.uuid(),
    key: boundedUnicodeString(256),
    value: requiredJSONSchema,
    createdAt: z.iso.datetime().transform((value) => new Date(value)),
  })
  .strict();

export type StepValueVersion = z.infer<typeof StepValueVersionSchema>;

/** One immutable Manuscript save. */
export const ManuscriptVersionSchema = z
  .object({
    id: z.uuid(),
    projectID: z.uuid(),
    value: ManuscriptValueSchema,
    createdAt: z.iso.datetime().transform((value) => new Date(value)),
  })
  .strict();

export type ManuscriptVersion = z.infer<typeof ManuscriptVersionSchema>;

/** Current convenience snapshot of every content kind saved under a Project. */
export const ProjectSchema = z
  .object({
    id: z.uuid(),
    createdAt: z.iso.datetime().transform((value) => new Date(value)),
    idea: IdeaVersionSchema,
    stepValues: z.array(StepValueVersionSchema),
    manuscript: ManuscriptVersionSchema.nullable(),
  })
  .strict();

export type Project = z.infer<typeof ProjectSchema>;

/** Complete initial value used to create a Project. */
export const ProjectCreateRequestSchema = z
  .object({
    idea: IdeaValueSchema,
  })
  .strict();

export type ProjectCreateRequest = z.infer<typeof ProjectCreateRequestSchema>;

/** Complete Idea Version saved under an existing Project. */
export const IdeaVersionCreateRequestSchema = z
  .object({
    projectID: z.uuid(),
    value: IdeaValueSchema,
  })
  .strict();

export type IdeaVersionCreateRequest = z.infer<typeof IdeaVersionCreateRequestSchema>;

/** Opaque Step Value saved under a client-controlled key. */
export const StepValueCreateRequestSchema = z
  .object({
    projectID: z.uuid(),
    key: boundedUnicodeString(256),
    value: requiredJSONSchema,
  })
  .strict();

export type StepValueCreateRequest = z.infer<typeof StepValueCreateRequestSchema>;

/** Complete Manuscript Version saved under an existing Project. */
export const ManuscriptCreateRequestSchema = z
  .object({
    projectID: z.uuid(),
    value: ManuscriptValueSchema,
  })
  .strict();

export type ManuscriptCreateRequest = z.infer<typeof ManuscriptCreateRequestSchema>;

/** Creates a Project and its initial Idea. */
export async function projectCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectCreateRequest
): Promise<Project> {
  return await api.fetch("/v0/projects", ProjectSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/** Reads the latest Idea, each latest Step Value, and the latest Manuscript. */
export async function projectGet(api: NarrativeEngineApi, accessToken: string, projectID: string): Promise<Project> {
  return await api.fetch(`/v0/projects/${projectID}`, ProjectSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/** Saves one complete Idea Version. */
export async function ideaVersionCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: IdeaVersionCreateRequest
): Promise<IdeaVersion> {
  return await api.fetch(`/v0/projects/${request.projectID}/ideas`, IdeaVersionSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ value: request.value }),
  });
}

/** Lists the retained Idea history, newest first. */
export async function ideaHistory(
  api: NarrativeEngineApi,
  accessToken: string,
  projectID: string
): Promise<IdeaVersion[]> {
  return await api.fetch(`/v0/projects/${projectID}/ideas`, z.array(IdeaVersionSchema).max(25), {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/** Saves arbitrary valid JSON without applying an Engine schema. */
export async function stepValueCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: StepValueCreateRequest
): Promise<StepValueVersion> {
  return await api.fetch(`/v0/projects/${request.projectID}/step-values`, StepValueVersionSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ key: request.key, value: request.value }),
  });
}

/** Lists retained Step Values for one opaque key, newest first. */
export async function stepValueHistory(
  api: NarrativeEngineApi,
  accessToken: string,
  projectID: string,
  key: string
): Promise<StepValueVersion[]> {
  const parameters = new URLSearchParams({ key });

  return await api.fetch(
    `/v0/projects/${projectID}/step-values?${parameters.toString()}`,
    z.array(StepValueVersionSchema).max(25),
    {
      method: "GET",
      headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    }
  );
}

/** Saves one complete Manuscript Version. */
export async function manuscriptCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ManuscriptCreateRequest
): Promise<ManuscriptVersion> {
  return await api.fetch(`/v0/projects/${request.projectID}/manuscripts`, ManuscriptVersionSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify({ value: request.value }),
  });
}

/** Lists the retained Manuscript history, newest first. */
export async function manuscriptHistory(
  api: NarrativeEngineApi,
  accessToken: string,
  projectID: string
): Promise<ManuscriptVersion[]> {
  return await api.fetch(`/v0/projects/${projectID}/manuscripts`, z.array(ManuscriptVersionSchema).max(25), {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}
