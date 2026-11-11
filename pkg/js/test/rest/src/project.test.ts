// @vitest-environment node
import { anonymousAccessToken, superAdminAccessToken, userAccessToken } from "./token";

import { readFileSync } from "node:fs";

import { describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  LayerKind,
  NarrativeEngineApi,
  PictureRefusal,
  type ProjectComposer,
  type ProjectCreateRequest,
  ProjectGenresSchema,
  ProjectTagsSchema,
  type ProjectUpdateRequest,
  beatCreate,
  cellWrite,
  projectCreate,
  projectDelete,
  projectGet,
  projectList,
  projectLogoDelete,
  projectLogoGet,
  projectLogoUpdate,
  projectStream,
  projectUpdate,
  sceneCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

/** A 16 px square WebP, a logo the server accepts. */
const logo = new Blob([readFileSync(new URL("../testdata/logo.webp", import.meta.url))], { type: "image/webp" });
/** A 64 by 36 px JPEG, a wide logo the server accepts. */
const wideLogo = new Blob([readFileSync(new URL("../testdata/wide-logo.jpg", import.meta.url))], {
  type: "image/jpeg",
});
/** A 16 by 129 px JPEG, a pixel taller than a logo may be. */
const tallLogo = new Blob([readFileSync(new URL("../testdata/tall-logo.jpg", import.meta.url))], {
  type: "image/jpeg",
});
/** A 16 px square PNG. */
const pngLogo = new Blob([readFileSync(new URL("../testdata/logo.png", import.meta.url))], { type: "image/png" });
/** A 16 px square PNG that declares an animation. */
const animatedLogo = new Blob([readFileSync(new URL("../testdata/animated-logo.png", import.meta.url))], {
  type: "image/png",
});

/** Lists the IDs of the caller's projects, most recently edited first. */
async function listedIDs(): Promise<string[]> {
  return (await projectList(api, accessToken, { limit: 100 })).projects.map((project) => project.id);
}

/** Reads a stream to its end. */
async function collect<T>(stream: AsyncIterable<T>): Promise<T[]> {
  const items: T[] = [];
  for await (const item of stream) items.push(item);
  return items;
}

/** Creates a project the test deletes when it finishes, whatever its outcome. */
async function createProject(request: ProjectCreateRequest): Promise<ProjectComposer> {
  const project = await projectCreate(api, accessToken, request);
  onTestFinished(() =>
    projectDelete(api, accessToken, project).then(
      () => undefined,
      () => undefined
    )
  );

  return project;
}

describe("ProjectTagsSchema", () => {
  it("accepts tags within the limits", () => {
    const tags = [
      "noir",
      "huis-clos",
      "années-50",
      "می\u200cخواهم",
      "क्\u200dष",
      "เรื่อง",
      "हिंदी",
      "ラーメン",
      "i\u0307stanbul",
      "é".repeat(24),
    ];
    expect(ProjectTagsSchema.parse(tags)).toEqual(tags);
    expect(ProjectTagsSchema.parse([])).toEqual([]);
    expect(ProjectTagsSchema.parse(Array.from({ length: 10 }, (_, index) => `tag-${index}`))).toHaveLength(10);
  });

  it("refuses tags the server refuses", () => {
    const refused = [
      [""],
      ["   "],
      [" noir"],
      ["noir "],
      ["Noir"],
      ["huis clos"],
      ["roman_maritime"],
      ["-noir"],
      ["noir-"],
      ["huis--clos"],
      ["anne\u0301es-50"],
      ["é".repeat(25)],
      ["noir", "noir"],
      Array.from({ length: 11 }, (_, index) => `tag-${index}`),
    ];

    for (const tags of refused) expect(ProjectTagsSchema.safeParse(tags).success).toBe(false);
  });
});

describe("ProjectGenresSchema", () => {
  it("accepts genres within the limits", () => {
    const genres = ["polar", "Polar", "roman noir", "é".repeat(24)];
    expect(ProjectGenresSchema.parse(genres)).toEqual(genres);
    expect(ProjectGenresSchema.parse([])).toEqual([]);
    expect(ProjectGenresSchema.parse(Array.from({ length: 10 }, (_, index) => `genre ${index}`))).toHaveLength(10);
  });

  it("refuses genres the server refuses", () => {
    const refused = [
      [""],
      ["   "],
      [" polar"],
      ["polar "],
      ["é".repeat(25)],
      ["polar", "polar"],
      Array.from({ length: 11 }, (_, index) => `genre ${index}`),
    ];

    for (const genres of refused) expect(ProjectGenresSchema.safeParse(genres).success).toBe(false);
  });
});

describe("projectCreate", () => {
  it("creates a project with its required layers", async () => {
    const project = await createProject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: Language.Fr,
    });

    expect(project).toMatchObject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: "fr",
      tags: [],
      genres: [],
    });
    expect(project.layers.map((layer) => [layer.position, layer.kind])).toEqual([[0, LayerKind.Manuscript]]);
  });

  it("stores the tags in order", async () => {
    const project = await createProject({ title: "Clear Hours", language: Language.En, tags: ["noir", "huis-clos"] });
    expect(project.tags).toEqual(["noir", "huis-clos"]);
  });

  it("stores the genres in order", async () => {
    const project = await createProject({ title: "Clear Hours", language: Language.En, genres: ["thriller", "polar"] });
    expect(project.genres).toEqual(["thriller", "polar"]);
  });

  it("names the invalid field of a blank title", async () => {
    await expect(projectCreate(api, accessToken, { title: "   ", language: Language.En })).rejects.toThrow(
      '"invalidFields":{"title":"notblank"}'
    );
  });

  it("names the invalid field of a title past 80 characters", async () => {
    await expect(projectCreate(api, accessToken, { title: "é".repeat(81), language: Language.En })).rejects.toThrow(
      '"invalidFields":{"title":"max"}'
    );
  });

  it("names the invalid field of a tag outside lower kebab case", async () => {
    await expect(
      projectCreate(api, accessToken, { title: "Clear Hours", language: Language.En, tags: ["huis clos"] })
    ).rejects.toThrow('"invalidFields":{"tags[0]":"projecttag"}');
  });

  it("names the invalid field of a genre with whitespace at an end", async () => {
    await expect(
      projectCreate(api, accessToken, { title: "Clear Hours", language: Language.En, genres: ["polar "] })
    ).rejects.toThrow('"invalidFields":{"genres[0]":"projectlabel"}');
  });

  it("names the invalid field of repeated genres", async () => {
    await expect(
      projectCreate(api, accessToken, { title: "Clear Hours", language: Language.En, genres: ["polar", "polar"] })
    ).rejects.toThrow('"invalidFields":{"genres":"unique"}');
  });

  it("names the invalid field of repeated tags", async () => {
    await expect(
      projectCreate(api, accessToken, { title: "Clear Hours", language: Language.En, tags: ["noir", "noir"] })
    ).rejects.toThrow('"invalidFields":{"tags":"unique"}');
  });

  it("returns 401 without a session", async () => {
    await expectStatus(projectCreate(api, "", { title: "Clear Hours", language: Language.En }), 401);
  });

  it("returns 403 for an anonymous session", async () => {
    const anonymous = await anonymousAccessToken();
    await expectStatus(projectCreate(api, anonymous, { title: "Clear Hours", language: Language.En }), 403);
  });
});

describe("projectGet", () => {
  it("returns the project", async () => {
    const created = await createProject({
      title: "Clear Hours",
      premise: "Marie waits.",
      language: Language.En,
    });

    const project = await projectGet(api, accessToken, { id: created.id });
    expect(project).toEqual({
      id: created.id,
      title: "Clear Hours",
      premise: "Marie waits.",
      language: Language.En,
      tags: [],
      genres: [],
      createdAt: created.createdAt,
      updatedAt: created.updatedAt,
      editedAt: created.editedAt,
    });
  });

  it("returns 400 for a malformed ID", async () => {
    await expectStatus(projectGet(api, accessToken, { id: "not-a-uuid" }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(projectGet(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("projectList", () => {
  it("lists projects newest first, a page at a time, with their total", async () => {
    const older = await createProject({ title: "Older", language: Language.En });
    const newer = await createProject({ title: "Newer", language: Language.En });

    const ids = (await projectList(api, accessToken)).projects.map((project) => project.id);
    expect(ids.indexOf(newer.id)).toBeGreaterThanOrEqual(0);
    expect(ids.indexOf(newer.id)).toBeLessThan(ids.indexOf(older.id));

    // Other test files create projects meanwhile, so the page after the newer project still holds the older one.
    const offset = ids.indexOf(newer.id) + 1;
    const nextPage = (await projectList(api, accessToken, { offset, limit: 100 })).projects.map(
      (project) => project.id
    );
    expect(nextPage).toContain(older.id);

    const firstPage = await projectList(api, accessToken, { limit: 1 });
    expect(firstPage.projects).toHaveLength(1);
    expect(firstPage.total).toBeGreaterThanOrEqual(2);
  });

  it("lists the project a cell write edits first", async () => {
    const edited = await createProject({ title: "Edited", language: Language.En });
    const beat = await beatCreate(api, accessToken, { projectID: edited.id, title: "Le départ", color: "#4a90d9" });
    const scene = await sceneCreate(api, accessToken, {
      projectID: edited.id,
      beatID: beat.id,
      position: 0,
      title: "Le quai",
    });
    const before = await projectGet(api, accessToken, { id: edited.id });
    const untouched = await createProject({ title: "Untouched", language: Language.En });

    let ids = await listedIDs();
    expect(ids.indexOf(untouched.id)).toBeGreaterThanOrEqual(0);
    expect(ids.indexOf(untouched.id)).toBeLessThan(ids.indexOf(edited.id));

    await cellWrite(api, accessToken, {
      projectID: edited.id,
      sceneID: scene.id,
      layerID: edited.layers[0].id,
      content: { blocks: [{ kind: "narration", text: "Le train entrait en gare." }] },
    });

    ids = await listedIDs();
    expect(ids.indexOf(edited.id)).toBeGreaterThanOrEqual(0);
    expect(ids.indexOf(edited.id)).toBeLessThan(ids.indexOf(untouched.id));

    // The cell write edits the project and leaves its own fields as they were.
    const after = await projectGet(api, accessToken, { id: edited.id });
    expect(after.editedAt.getTime()).toBeGreaterThan(before.editedAt.getTime());
    expect(after.updatedAt).toEqual(before.updatedAt);
  });

  it("returns 400 for a page over 100 projects", async () => {
    await expectStatus(projectList(api, accessToken, { limit: 101 }), 400);
  });

  it("returns 400 for a negative offset", async () => {
    await expectStatus(projectList(api, accessToken, { offset: -1 }), 400);
  });
});

describe("projectUpdate", () => {
  it("replaces the fields", async () => {
    const created = await createProject({ title: "Draft", language: Language.En });

    const project = await projectUpdate(api, accessToken, {
      id: created.id,
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: Language.Fr,
      tags: ["noir", "huis-clos"],
      genres: ["thriller"],
    });
    expect(project).toMatchObject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: "fr",
      tags: ["noir", "huis-clos"],
      genres: ["thriller"],
    });
  });

  it("replaces the whole tag list, and clears it with an empty one", async () => {
    const created = await createProject({ title: "Draft", language: Language.En, tags: ["noir", "polar"] });
    const fields = { id: created.id, title: "Draft", language: Language.En, genres: [] };

    const replaced = await projectUpdate(api, accessToken, { ...fields, tags: ["huis-clos"] });
    expect(replaced.tags).toEqual(["huis-clos"]);

    const cleared = await projectUpdate(api, accessToken, { ...fields, tags: [] });
    expect(cleared.tags).toEqual([]);
  });

  it("names the omitted tags and keeps the project's list", async () => {
    const created = await createProject({ title: "Draft", language: Language.En, tags: ["noir"] });
    const request = {
      id: created.id,
      title: "Renamed",
      language: Language.En,
      genres: [],
    } as unknown as ProjectUpdateRequest;

    await expect(projectUpdate(api, accessToken, request)).rejects.toThrow('"invalidFields":{"tags":"required"}');
    expect(await projectGet(api, accessToken, { id: created.id })).toMatchObject({ title: "Draft", tags: ["noir"] });
  });

  it("replaces the whole genre list, and clears it with an empty one", async () => {
    const created = await createProject({ title: "Draft", language: Language.En, genres: ["polar", "thriller"] });
    const fields = { id: created.id, title: "Draft", language: Language.En, tags: [] };

    const replaced = await projectUpdate(api, accessToken, { ...fields, genres: ["roman noir"] });
    expect(replaced.genres).toEqual(["roman noir"]);

    const cleared = await projectUpdate(api, accessToken, { ...fields, genres: [] });
    expect(cleared.genres).toEqual([]);
  });

  it("names the omitted genres and keeps the project's list", async () => {
    const created = await createProject({ title: "Draft", language: Language.En, genres: ["polar"] });
    const request = {
      id: created.id,
      title: "Renamed",
      language: Language.En,
      tags: [],
    } as unknown as ProjectUpdateRequest;

    await expect(projectUpdate(api, accessToken, request)).rejects.toThrow('"invalidFields":{"genres":"required"}');
    expect(await projectGet(api, accessToken, { id: created.id })).toMatchObject({ title: "Draft", genres: ["polar"] });
  });

  it("returns 404 for an absent project", async () => {
    const request = { id: ABSENT_ID, title: "Renamed", language: Language.En, tags: [], genres: [] };
    await expectStatus(projectUpdate(api, accessToken, request), 404);
  });
});

describe("projectDelete", () => {
  it("deletes the project", async () => {
    const created = await createProject({ title: "Doomed", language: Language.En });

    await expect(projectDelete(api, accessToken, created)).resolves.toMatchObject({ id: created.id });
    await expectStatus(projectGet(api, accessToken, { id: created.id }), 404);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(projectDelete(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("projectStream", () => {
  it("streams the project, then its beats, scenes, layers and cells", async () => {
    const project = await createProject({ title: "Les Heures claires", language: Language.Fr });
    const beat = await beatCreate(api, accessToken, { projectID: project.id, title: "Le départ", color: "#4a90d9" });
    const scene = await sceneCreate(api, accessToken, {
      projectID: project.id,
      beatID: beat.id,
      position: 0,
      title: "Le quai",
    });
    const layer = project.layers[0];
    await cellWrite(api, accessToken, {
      projectID: project.id,
      sceneID: scene.id,
      layerID: layer.id,
      content: { blocks: [{ kind: "narration", text: "Le train entrait en gare." }] },
    });

    const records = await collect(projectStream(api, accessToken, { id: project.id }));

    expect(records).toMatchObject([
      { type: "project", project: { id: project.id, title: "Les Heures claires" } },
      { type: "beat", beat: { id: beat.id } },
      { type: "scene", scene: { id: scene.id } },
      { type: "layer", layer: { id: layer.id } },
      { type: "cell", cell: { sceneID: scene.id, layerID: layer.id } },
    ]);
  });

  it("returns 400 for a malformed ID", async () => {
    await expectStatus(collect(projectStream(api, accessToken, { id: "not-a-uuid" })), 400);
  });

  it("returns 401 without a session", async () => {
    await expectStatus(collect(projectStream(api, "", { id: ABSENT_ID })), 401);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(collect(projectStream(api, accessToken, { id: ABSENT_ID })), 404);
  });
});

describe("projectLogoUpdate", () => {
  it("sets the logo, with a version, and keeps the project's edit date", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });

    const project = await projectLogoUpdate(api, accessToken, { projectID: created.id }, logo);
    expect(project.logoVersion).toEqual(expect.any(String));
    expect(project.editedAt).toEqual(created.editedAt);
    expect(project.updatedAt).toEqual(created.updatedAt);
  });

  it("sets a logo of any aspect ratio", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });

    const project = await projectLogoUpdate(api, accessToken, { projectID: created.id }, wideLogo);
    expect(project.logoVersion).toEqual(expect.any(String));
  });

  it("sets a still PNG", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });

    const project = await projectLogoUpdate(api, accessToken, { projectID: created.id }, pngLogo);
    expect(project.logoVersion).toEqual(expect.any(String));
    expect((await projectLogoGet(api, accessToken, { projectID: created.id })).type).toBe("image/png");
  });

  it("keeps the version on every read and write of the project", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const pictured = await projectLogoUpdate(api, accessToken, { projectID: created.id }, logo);
    const version = { logoVersion: pictured.logoVersion };

    expect(await projectGet(api, accessToken, { id: created.id })).toMatchObject(version);
    const listed = (await projectList(api, accessToken, { limit: 100 })).projects.find(
      (project) => project.id === created.id
    );
    expect(listed).toMatchObject(version);
    const updated = await projectUpdate(api, accessToken, {
      id: created.id,
      title: "Renamed",
      language: Language.En,
      tags: [],
      genres: [],
    });
    expect(updated).toMatchObject(version);
    expect(await projectDelete(api, accessToken, created)).toMatchObject(version);
  });

  it("replaces a logo with a new version", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id };

    const first = await projectLogoUpdate(api, accessToken, request, logo);
    const second = await projectLogoUpdate(api, accessToken, request, logo);
    expect(second.logoVersion).not.toBe(first.logoVersion);
  });

  it("returns 422 with the reason for a refused logo", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const refusals = [
      { body: tallLogo, reason: PictureRefusal.Dimensions },
      { body: new Blob(["not an image"], { type: "image/webp" }), reason: PictureRefusal.Format },
      { body: animatedLogo, reason: PictureRefusal.Format },
    ];

    for (const { body, reason } of refusals) {
      const update = projectLogoUpdate(api, accessToken, { projectID: created.id }, body);
      await expect(update).rejects.toThrow(`"reason":"${reason}"`);
      await expectStatus(update, 422);
    }
  });

  it("returns 422 naming the size past the logo's bytes", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const body = new Blob([new Uint8Array(256 * 1024 + 1)], { type: "image/webp" });
    const update = projectLogoUpdate(api, accessToken, { projectID: created.id }, body);
    await expect(update).rejects.toThrow(`"reason":"${PictureRefusal.Size}"`);
    await expectStatus(update, 422);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(projectLogoUpdate(api, accessToken, { projectID: ABSENT_ID }, logo), 404);
  });

  it("returns 403 for an anonymous session", async () => {
    await expectStatus(projectLogoUpdate(api, await anonymousAccessToken(), { projectID: ABSENT_ID }, logo), 403);
  });
});

describe("projectLogoGet", () => {
  it("returns the logo as it was sent", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id };
    await projectLogoUpdate(api, accessToken, request, wideLogo);

    const stored = await projectLogoGet(api, accessToken, request);
    expect(stored.type).toBe("image/jpeg");
    expect(new Uint8Array(await stored.arrayBuffer())).toEqual(new Uint8Array(await wideLogo.arrayBuffer()));
  });

  it("returns 404 for a project without a logo", async () => {
    const created = await createProject({ title: "Bare", language: Language.En });
    await expectStatus(projectLogoGet(api, accessToken, { projectID: created.id }), 404);
  });

  it("returns 404 for the project of another user", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id };
    await projectLogoUpdate(api, accessToken, request, logo);

    await expectStatus(projectLogoGet(api, await userAccessToken(), request), 404);
  });

  it("returns 400 for a malformed project ID", async () => {
    await expectStatus(projectLogoGet(api, accessToken, { projectID: "not-a-uuid" }), 400);
  });
});

describe("projectLogoDelete", () => {
  it("removes the logo and keeps the project's edit date", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id };
    await projectLogoUpdate(api, accessToken, request, logo);

    const project = await projectLogoDelete(api, accessToken, request);
    expect(project.logoVersion).toBeUndefined();
    expect(project.editedAt).toEqual(created.editedAt);
    expect(project.updatedAt).toEqual(created.updatedAt);
    await expectStatus(projectLogoGet(api, accessToken, request), 404);
  });

  it("keeps a project without a logo as it is", async () => {
    const created = await createProject({ title: "Bare", language: Language.En });
    const project = await projectLogoDelete(api, accessToken, { projectID: created.id });
    expect(project).toMatchObject({ id: created.id, title: "Bare" });
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(projectLogoDelete(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
