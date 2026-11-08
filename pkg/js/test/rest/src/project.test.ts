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
  ProjectPicture,
  beatCreate,
  cellWrite,
  projectCreate,
  projectDelete,
  projectGet,
  projectList,
  projectPictureDelete,
  projectPictureGet,
  projectPictureUpdate,
  projectStream,
  projectUpdate,
  sceneCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

/** A 16 px square WebP, a logo the server accepts. */
const logo = new Blob([readFileSync(new URL("../testdata/logo.webp", import.meta.url))], { type: "image/webp" });
/** A 64 by 36 px JPEG, a 16:9 background the server accepts. */
const background = new Blob([readFileSync(new URL("../testdata/background.jpg", import.meta.url))], {
  type: "image/jpeg",
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
    });
    expect(project.layers.map((layer) => [layer.position, layer.kind])).toEqual([[0, LayerKind.Manuscript]]);
  });

  it("names the invalid field of a blank title", async () => {
    await expect(projectCreate(api, accessToken, { title: "   ", language: Language.En })).rejects.toThrow(
      '"invalidFields":{"title":"notblank"}'
    );
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
    });
    expect(project).toMatchObject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: "fr",
    });
  });

  it("returns 404 for an absent project", async () => {
    const request = { id: ABSENT_ID, title: "Renamed", language: Language.En };
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

describe("projectPictureUpdate", () => {
  it("sets the logo, with a version, as an edit of the project", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });

    const project = await projectPictureUpdate(
      api,
      accessToken,
      { projectID: created.id, picture: ProjectPicture.Logo },
      logo
    );
    expect(project.logoVersion).toEqual(expect.any(String));
    expect(project.backgroundVersion).toBeUndefined();
    expect(project.editedAt.getTime()).toBeGreaterThan(created.editedAt.getTime());
    expect(project.updatedAt).toEqual(created.updatedAt);
  });

  it("sets the background beside the logo", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const withLogo = await projectPictureUpdate(
      api,
      accessToken,
      { projectID: created.id, picture: ProjectPicture.Logo },
      logo
    );

    const project = await projectPictureUpdate(
      api,
      accessToken,
      { projectID: created.id, picture: ProjectPicture.Background },
      background
    );
    expect(project.logoVersion).toBe(withLogo.logoVersion);
    expect(project.backgroundVersion).toEqual(expect.any(String));
  });

  it("replaces a picture with a new version", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id, picture: ProjectPicture.Logo };

    const first = await projectPictureUpdate(api, accessToken, request, logo);
    const second = await projectPictureUpdate(api, accessToken, request, logo);
    expect(second.logoVersion).not.toBe(first.logoVersion);
  });

  it("returns 422 with the reason for a refused picture", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const refusals = [
      { picture: ProjectPicture.Logo, body: background, reason: PictureRefusal.Ratio },
      { picture: ProjectPicture.Background, body: logo, reason: PictureRefusal.Ratio },
      {
        picture: ProjectPicture.Logo,
        body: new Blob(["not an image"], { type: "image/webp" }),
        reason: PictureRefusal.Format,
      },
      // A background may hold more bytes than a logo, so the server reads them.
      {
        picture: ProjectPicture.Background,
        body: new Blob([new Uint8Array(256 * 1024 + 1)], { type: "image/webp" }),
        reason: PictureRefusal.Format,
      },
    ];

    for (const { picture, body, reason } of refusals) {
      const update = projectPictureUpdate(api, accessToken, { projectID: created.id, picture }, body);
      await expect(update).rejects.toThrow(`"reason":"${reason}"`);
      await expectStatus(update, 422);
    }
  });

  it("returns 400 naming the data past the logo's bytes", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const body = new Blob([new Uint8Array(256 * 1024 + 1)], { type: "image/webp" });
    const update = projectPictureUpdate(
      api,
      accessToken,
      { projectID: created.id, picture: ProjectPicture.Logo },
      body
    );
    await expect(update).rejects.toThrow('"invalidFields":{"data":"picturebytes"}');
    await expectStatus(update, 400);
  });

  it("returns 400 naming an unknown picture", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id, picture: "banner" as ProjectPicture };
    const update = projectPictureUpdate(api, accessToken, request, logo);
    await expect(update).rejects.toThrow('"picture":"projectpicture"');
    await expectStatus(update, 400);
  });

  it("returns 404 for an absent project", async () => {
    const request = { projectID: ABSENT_ID, picture: ProjectPicture.Logo };
    await expectStatus(projectPictureUpdate(api, accessToken, request, logo), 404);
  });

  it("returns 403 for an anonymous session", async () => {
    const request = { projectID: ABSENT_ID, picture: ProjectPicture.Logo };
    await expectStatus(projectPictureUpdate(api, await anonymousAccessToken(), request, logo), 403);
  });
});

describe("projectPictureGet", () => {
  it("returns the picture as it was sent", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id, picture: ProjectPicture.Background };
    await projectPictureUpdate(api, accessToken, request, background);

    const picture = await projectPictureGet(api, accessToken, request);
    expect(picture.type).toBe("image/jpeg");
    expect(new Uint8Array(await picture.arrayBuffer())).toEqual(new Uint8Array(await background.arrayBuffer()));
  });

  it("returns 404 for a project without the picture", async () => {
    const created = await createProject({ title: "Bare", language: Language.En });
    await expectStatus(
      projectPictureGet(api, accessToken, { projectID: created.id, picture: ProjectPicture.Logo }),
      404
    );
  });

  it("returns 404 for the project of another user", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id, picture: ProjectPicture.Logo };
    await projectPictureUpdate(api, accessToken, request, logo);

    await expectStatus(projectPictureGet(api, await userAccessToken(), request), 404);
  });

  it("returns 400 for a malformed project ID", async () => {
    await expectStatus(
      projectPictureGet(api, accessToken, { projectID: "not-a-uuid", picture: ProjectPicture.Logo }),
      400
    );
  });
});

describe("projectPictureDelete", () => {
  it("removes the picture, as an edit of the project", async () => {
    const created = await createProject({ title: "Pictured", language: Language.En });
    const request = { projectID: created.id, picture: ProjectPicture.Logo };
    const pictured = await projectPictureUpdate(api, accessToken, request, logo);

    const project = await projectPictureDelete(api, accessToken, request);
    expect(project.logoVersion).toBeUndefined();
    expect(project.editedAt.getTime()).toBeGreaterThan(pictured.editedAt.getTime());
    await expectStatus(projectPictureGet(api, accessToken, request), 404);
  });

  it("keeps a project without the picture as it is", async () => {
    const created = await createProject({ title: "Bare", language: Language.En });
    const project = await projectPictureDelete(api, accessToken, {
      projectID: created.id,
      picture: ProjectPicture.Background,
    });
    expect(project).toMatchObject({ id: created.id, title: "Bare" });
    expect(project.backgroundVersion).toBeUndefined();
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(
      projectPictureDelete(api, accessToken, { projectID: ABSENT_ID, picture: ProjectPicture.Logo }),
      404
    );
  });
});
