import { anonymousAccessToken, superAdminAccessToken } from "./token";

import { describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  LayerKind,
  NarrativeEngineApi,
  type ProjectComposer,
  type ProjectCreateRequest,
  beatCreate,
  cellWrite,
  projectCreate,
  projectDelete,
  projectGet,
  projectList,
  projectStream,
  projectUpdate,
  sceneCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

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
