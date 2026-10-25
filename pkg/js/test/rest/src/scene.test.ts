import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  projectCreate,
  projectDelete,
  projectGet,
  sceneCreate,
  sceneDelete,
  sceneMove,
  sceneUpdate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

let project: ProjectComposer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: "Scenes", language: Language.En });
});

afterAll(async () => {
  await projectDelete(api, accessToken, await projectGet(api, accessToken, project));
});

/** Returns the scene titles of the test project, in narrative order. */
async function sceneTitles(): Promise<string[]> {
  return (await projectGet(api, accessToken, project)).scenes.map((scene) => scene.title);
}

/** Deletes every scene the test created. */
async function deleteScenes(...scenes: Scene[]): Promise<void> {
  for (const scene of scenes) {
    const stored = (await projectGet(api, accessToken, project)).scenes.find((item) => item.id === scene.id);
    if (stored) await sceneDelete(api, accessToken, stored);
  }
}

describe("sceneCreate", () => {
  it("places a scene first, or after the scene it names", async () => {
    const second = await sceneCreate(api, accessToken, { projectID: project.id, title: "Second" });
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First" });
    const third = await sceneCreate(api, accessToken, { projectID: project.id, after: second.id, title: "Third" });

    expect(first.position).toBe(0);
    expect(third.position).toBe(2);
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Third"]);

    await deleteScenes(first, second, third);
  });

  it("returns 400 for an anchor outside the project", async () => {
    await expectStatus(sceneCreate(api, accessToken, { projectID: project.id, after: ABSENT_ID }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneCreate(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});

describe("sceneUpdate", () => {
  it("replaces the fields and raises the revision", async () => {
    const created = await sceneCreate(api, accessToken, { projectID: project.id, title: "Draft" });

    const scene = await sceneUpdate(api, accessToken, {
      id: created.id,
      revision: created.revision,
      title: "Ouverture",
      synopsis: "Marie attend.",
    });
    expect(scene).toMatchObject({ title: "Ouverture", synopsis: "Marie attend.", revision: 2 });

    await deleteScenes(scene);
  });

  it("returns 409 for a stale revision", async () => {
    const created = await sceneCreate(api, accessToken, { projectID: project.id });
    await sceneUpdate(api, accessToken, { id: created.id, revision: created.revision, title: "Renamed" });

    await expectStatus(sceneUpdate(api, accessToken, { id: created.id, revision: created.revision }), 409);

    await deleteScenes(created);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneUpdate(api, accessToken, { id: ABSENT_ID, revision: 1 }), 404);
  });
});

describe("sceneMove", () => {
  it("moves a scene and leaves the revisions alone", async () => {
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First" });
    const second = await sceneCreate(api, accessToken, { projectID: project.id, after: first.id, title: "Second" });

    const scenes = await sceneMove(api, accessToken, { id: first.id, after: second.id });
    expect(scenes.map((scene) => [scene.position, scene.title, scene.revision])).toEqual([
      [0, "Second", 1],
      [1, "First", 1],
    ]);

    const back = await sceneMove(api, accessToken, { id: first.id });
    expect(back.map((scene) => scene.title)).toEqual(["First", "Second"]);

    await deleteScenes(first, second);
  });

  it("keeps positions contiguous under concurrent moves", async () => {
    const scenes: Scene[] = [];
    for (let i = 0; i < 6; i++) {
      scenes.push(await sceneCreate(api, accessToken, { projectID: project.id, after: scenes.at(-1)?.id }));
    }

    await Promise.all(scenes.map((scene) => sceneMove(api, accessToken, { id: scene.id })));

    const stored = (await projectGet(api, accessToken, project)).scenes;
    expect(stored.map((scene) => scene.position)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(stored.map((scene) => scene.id))).toEqual(new Set(scenes.map((scene) => scene.id)));

    await deleteScenes(...scenes);
  });

  it("returns 400 for a scene placed after itself", async () => {
    const scene = await sceneCreate(api, accessToken, { projectID: project.id });

    await expectStatus(sceneMove(api, accessToken, { id: scene.id, after: scene.id }), 400);

    await deleteScenes(scene);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneMove(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("sceneDelete", () => {
  it("deletes the scene and closes the gap", async () => {
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First" });
    const second = await sceneCreate(api, accessToken, { projectID: project.id, after: first.id, title: "Second" });

    await expect(sceneDelete(api, accessToken, first)).resolves.toMatchObject({ id: first.id });

    const scenes = (await projectGet(api, accessToken, project)).scenes;
    expect(scenes.map((scene) => [scene.position, scene.title])).toEqual([[0, "Second"]]);

    await deleteScenes(second);
  });

  it("returns 409 for a stale revision", async () => {
    const created = await sceneCreate(api, accessToken, { projectID: project.id });
    await sceneUpdate(api, accessToken, { id: created.id, revision: created.revision, title: "Renamed" });

    await expectStatus(sceneDelete(api, accessToken, created), 409);

    await deleteScenes(created);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneDelete(api, accessToken, { id: ABSENT_ID, revision: 1 }), 404);
  });
});
