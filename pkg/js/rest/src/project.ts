import type { NarrativeEngineApi } from "./api";
import { type Beat, BeatSchema } from "./beat";
import { type Cell, CellSchema } from "./cell";
import { type Layer, LayerSchema } from "./layer";
import { type Scene, SceneSchema } from "./scene";
import { maxChars } from "./text";

import { HTTP_HEADERS } from "@a-novel-kit/nodelib-browser/http";

import { z } from "zod";

/** The languages a project is written in, as BCP 47 tags. */
export enum Language {
  /** English. */
  En = "en",
  /** French. */
  Fr = "fr",
}

/** Validates a {@link Language}. */
export const LanguageSchema = z.enum([Language.En, Language.Fr]);

/** What a logo accepts: any aspect ratio, at most `maxWidth` pixels wide and `maxHeight` tall, and `maxBytes` encoded. */
export const PROJECT_LOGO_RULES = { maxHeight: 128, maxBytes: 262144 } as const;

/** The rules a logo breaks when the server refuses it, as a 422 error's `tags.reason` names them. */
export enum PictureRefusal {
  /** The bytes are neither a still WebP, a JPEG nor a still PNG, or do not decode whole. */
  Format = "format",
  /** The logo is under 1 pixel on a side, or past its tallest. */
  Dimensions = "dimensions",
  /** The logo is past its largest size, in bytes. */
  Size = "size",
}

/** One narrative, written in one language, that its owner composes scene by scene. */
export const ProjectSchema = z.object({
  id: z.uuid(),
  title: z.string(),
  premise: z.string(),
  language: LanguageSchema,
  /** The author's custom labels, in order. Empty until the author adds one. */
  tags: z.array(z.string()),
  /** The genres the author names, in order. Empty until the author adds one. */
  genres: z.array(z.string()),
  createdAt: z.iso.datetime().transform((value) => new Date(value)),
  /** When the project's own fields, its title, premise, language, tags and genres, last changed. */
  updatedAt: z.iso.datetime().transform((value) => new Date(value)),
  /** When the project's content last changed, its logo aside. {@link projectList} orders projects by it. */
  editedAt: z.iso.datetime().transform((value) => new Date(value)),
  /**
   * When the logo last changed, absent while the project has no logo. A logo URL that carries it names one
   * version, which a cache keeps for good. It stays the server's string: a Date would drop its microseconds.
   */
  logoVersion: z.iso.datetime().optional(),
});

/** The value {@link ProjectSchema} validates. */
export type Project = z.infer<typeof ProjectSchema>;

/** A new project with its required layers, in order, as {@link projectCreate} returns it. */
export const ProjectComposerSchema = ProjectSchema.extend({
  layers: z.array(LayerSchema),
});

/** The value {@link ProjectComposerSchema} validates. */
export type ProjectComposer = z.infer<typeof ProjectComposerSchema>;

/** A page of the caller's projects, with how many the caller holds in all. */
export const ProjectPageSchema = z.object({
  projects: z.array(ProjectSchema),
  total: z.int(),
});

/** The value {@link ProjectPageSchema} validates. */
export type ProjectPage = z.infer<typeof ProjectPageSchema>;

/** The title of a project, which is not blank. */
export const ProjectTitleSchema = maxChars(80).regex(/\S/);

/** What a project's narrative is about, which may be empty. */
export const ProjectPremiseSchema = maxChars(20000);

/** The most tags a project holds. */
export const PROJECT_TAGS_MAX = 10;

/** The most characters a tag of a project holds. */
export const PROJECT_TAG_CHARS_MAX = 24;

/**
 * A tag of a project in lower kebab case, as `années-50` is: runs of lowercase letters or digits, from any script,
 * joined by single hyphens. An accented letter must be precomposed, as `normalize("NFC")` returns it.
 */
export const ProjectTagSchema = maxChars(PROJECT_TAG_CHARS_MAX).regex(
  /^[\p{Ll}\p{Lo}\p{Nd}]+(-[\p{Ll}\p{Lo}\p{Nd}]+)*$/u
);

/** The tags of a project, in order, each appearing once. */
export const ProjectTagsSchema = z
  .array(ProjectTagSchema)
  .max(PROJECT_TAGS_MAX)
  .refine((tags) => new Set(tags).size === tags.length, { error: "Expected every tag to appear once" });

/** The most genres a project holds. */
export const PROJECT_GENRES_MAX = 10;

/** The most characters a genre of a project holds. */
export const PROJECT_GENRE_CHARS_MAX = 24;

/** A genre of a project, at least one character long, with no whitespace at either end. */
export const ProjectGenreSchema = maxChars(PROJECT_GENRE_CHARS_MAX).regex(/^\S(?:.*\S)?$/s);

/** The genres of a project, in order, each appearing once. */
export const ProjectGenresSchema = z
  .array(ProjectGenreSchema)
  .max(PROJECT_GENRES_MAX)
  .refine((genres) => new Set(genres).size === genres.length, { error: "Expected every genre to appear once" });

/** The project to create. Its scenes come next, from `sceneCreate`. */
export const ProjectCreateRequestSchema = z.object({
  title: ProjectTitleSchema,
  premise: ProjectPremiseSchema.optional(),
  language: LanguageSchema,
  tags: ProjectTagsSchema.optional(),
  genres: ProjectGenresSchema.optional(),
});

/** The value {@link ProjectCreateRequestSchema} validates. */
export type ProjectCreateRequest = z.infer<typeof ProjectCreateRequestSchema>;

/** The project to read. */
export const ProjectGetRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link ProjectGetRequestSchema} validates. */
export type ProjectGetRequest = z.infer<typeof ProjectGetRequestSchema>;

/** A page of projects, most recently edited first: at most `limit`, 20 when omitted, after the first `offset`. */
export const ProjectListRequestSchema = z.object({
  limit: z.int().min(0).max(100).optional(),
  offset: z.int().min(0).optional(),
});

/** The value {@link ProjectListRequestSchema} validates. */
export type ProjectListRequest = z.infer<typeof ProjectListRequestSchema>;

/** The new fields of a project. An omitted premise is cleared. `tags` and `genres` each replace their whole list. */
export const ProjectUpdateRequestSchema = z.object({
  id: z.uuid(),
  title: ProjectTitleSchema,
  premise: ProjectPremiseSchema.optional(),
  language: LanguageSchema,
  tags: ProjectTagsSchema,
  genres: ProjectGenresSchema,
});

/** The value {@link ProjectUpdateRequestSchema} validates. */
export type ProjectUpdateRequest = z.infer<typeof ProjectUpdateRequestSchema>;

/** The project to delete. */
export const ProjectDeleteRequestSchema = z.object({
  id: z.uuid(),
});

/** The value {@link ProjectDeleteRequestSchema} validates. */
export type ProjectDeleteRequest = z.infer<typeof ProjectDeleteRequestSchema>;

/**
 * Creates a project with its required layers, and returns it with them. Throws on HTTP 409, with `projects` in
 * the error's `tags.limit`, when the caller already owns 100 projects.
 */
export async function projectCreate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectCreateRequest
): Promise<ProjectComposer> {
  return await api.fetch("/v1/projects", ProjectComposerSchema, {
    method: "POST",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Returns a project of the caller. Read its scenes, layers and cells page by page with `sceneList`,
 * `layerList` and `cellList`. Throws on HTTP 404 when the caller owns no such project.
 */
export async function projectGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectGetRequest
): Promise<Project> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/project?${params.toString()}`, ProjectSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Lists a page of the caller's projects, most recently edited first, with their total. Raise `offset` by the page
 * size for the next page.
 */
export async function projectList(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectListRequest = {}
): Promise<ProjectPage> {
  const params = new URLSearchParams();
  if (request.limit) params.set("limit", `${request.limit}`);
  if (request.offset) params.set("offset", `${request.offset}`);
  const query = params.toString();
  return await api.fetch(`/v1/projects${query ? `?${query}` : ""}`, ProjectPageSchema, {
    method: "GET",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Replaces the title, premise, language, tags and genres of a project. Throws on HTTP 404 for an unknown project, and 409
 * with `projectBytes` in the error's `tags.limit` when the project would pass its 16 MiB budget of stored bytes.
 */
export async function projectUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectUpdateRequest
): Promise<Project> {
  return await api.fetch("/v1/project", ProjectSchema, {
    method: "PUT",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
    body: JSON.stringify(request),
  });
}

/**
 * Deletes a project with its scenes, layers and cells, and returns it as it was. Throws on HTTP 404
 * for an unknown project.
 */
export async function projectDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectDeleteRequest
): Promise<Project> {
  const params = new URLSearchParams({ id: request.id });
  return await api.fetch(`/v1/project?${params.toString()}`, ProjectSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/** The project whose logo to replace, the encoded image aside. */
export const ProjectLogoUpdateRequestSchema = z.object({
  projectID: z.uuid(),
});

/** The value {@link ProjectLogoUpdateRequestSchema} validates. */
export type ProjectLogoUpdateRequest = z.infer<typeof ProjectLogoUpdateRequestSchema>;

/** The project whose logo to read. */
export const ProjectLogoGetRequestSchema = z.object({
  projectID: z.uuid(),
});

/** The value {@link ProjectLogoGetRequestSchema} validates. */
export type ProjectLogoGetRequest = z.infer<typeof ProjectLogoGetRequestSchema>;

/** The project whose logo to remove. */
export const ProjectLogoDeleteRequestSchema = z.object({
  projectID: z.uuid(),
});

/** The value {@link ProjectLogoDeleteRequestSchema} validates. */
export type ProjectLogoDeleteRequest = z.infer<typeof ProjectLogoDeleteRequestSchema>;

/**
 * Replaces the logo of a project with `body`, an encoded still WebP, JPEG or still PNG, and returns the project. A
 * logo is cosmetic, so the project keeps its `editedAt`. The server reads the format from the bytes and checks the
 * logo against {@link PROJECT_LOGO_RULES}. Throws on HTTP 422, with a {@link PictureRefusal} in the error's
 * `tags.reason`, for a logo it refuses, past `maxBytes` included, and on 404 when the caller owns no such project.
 */
export async function projectLogoUpdate(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectLogoUpdateRequest,
  body: Blob
): Promise<Project> {
  const params = new URLSearchParams({ projectID: request.projectID });
  return await api.fetch(`/v1/project/logo?${params.toString()}`, ProjectSchema, {
    method: "PUT",
    headers: { "Content-Type": body.type, Authorization: `Bearer ${accessToken}` },
    body,
  });
}

/**
 * Returns the logo of a project, typed with the media type it was stored with. Throws on HTTP 404 when the caller
 * owns no such project, or the project holds no logo.
 */
export async function projectLogoGet(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectLogoGetRequest
): Promise<Blob> {
  const params = new URLSearchParams({ projectID: request.projectID });
  return await api.fetchBlob(`/v1/project/logo?${params.toString()}`, {
    method: "GET",
    headers: { Authorization: `Bearer ${accessToken}` },
  });
}

/**
 * Removes the logo of a project, and returns the project, which keeps its `editedAt`. A project without a logo stays
 * as it is. Throws on HTTP 404 when the caller owns no such project.
 */
export async function projectLogoDelete(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectLogoDeleteRequest
): Promise<Project> {
  const params = new URLSearchParams({ projectID: request.projectID });
  return await api.fetch(`/v1/project/logo?${params.toString()}`, ProjectSchema, {
    method: "DELETE",
    headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${accessToken}` },
  });
}

/** A record of {@link projectStream}: the project, or one of its beats, scenes, layers or cells. */
export type ProjectStreamRecord =
  | { type: "project"; project: Project }
  | { type: "beat"; beat: Beat }
  | { type: "scene"; scene: Scene }
  | { type: "layer"; layer: Layer }
  | { type: "cell"; cell: Cell };

/** Options of {@link projectStream}. */
export type ProjectStreamOptions = {
  /** Stops the stream. */
  signal?: AbortSignal;
};

/**
 * Streams a project of the caller whole, in one response: the project, then its beats and scenes in narrative order,
 * its layers in order, and its cells by the narrative order of their scene, then the position of their layer, each
 * as its list returns it. A cut stream resumes after the last record received. A write during the stream may show,
 * as it does across the pages of a list. Throws on HTTP 404 when the caller owns no such project.
 */
export async function* projectStream(
  api: NarrativeEngineApi,
  accessToken: string,
  request: ProjectGetRequest,
  options: ProjectStreamOptions = {}
): AsyncGenerator<ProjectStreamRecord> {
  const params = new URLSearchParams({ id: request.id });
  const events = api.stream(
    `/v1/project/stream?${params.toString()}`,
    { method: "GET", headers: { Authorization: `Bearer ${accessToken}` }, signal: options.signal },
    (event) => event.type === "end"
  );

  for await (const event of events) {
    const data: unknown = JSON.parse(event.data);

    switch (event.type) {
      case "project":
        yield { type: "project", project: ProjectSchema.parse(data) };
        break;
      case "beat":
        yield { type: "beat", beat: BeatSchema.parse(data) };
        break;
      case "scene":
        yield { type: "scene", scene: SceneSchema.parse(data) };
        break;
      case "layer":
        yield { type: "layer", layer: LayerSchema.parse(data) };
        break;
      case "cell":
        yield { type: "cell", cell: CellSchema.parse(data) };
        break;
    }
  }
}
