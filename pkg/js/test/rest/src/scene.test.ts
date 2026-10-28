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
  sceneCreate,
  sceneDelete,
  sceneList,
  sceneMove,
  sceneUpdate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const COLOR = "#4a90d9";

let project: ProjectComposer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: "Scenes", language: Language.En });
});

afterAll(async () => {
  await projectDelete(api, accessToken, project);
});

/** Returns every scene of the test project, in narrative order. */
async function storedScenes(): Promise<Scene[]> {
  return (await sceneList(api, accessToken, { projectID: project.id, limit: 500 })).scenes;
}

/** Returns the scene titles of the test project, in narrative order. */
async function sceneTitles(): Promise<string[]> {
  return (await storedScenes()).map((scene) => scene.title);
}

/** Deletes every scene the test created. */
async function deleteScenes(...scenes: Scene[]): Promise<void> {
  for (const scene of scenes) {
    const stored = (await storedScenes()).find((item) => item.id === scene.id);
    if (stored) await sceneDelete(api, accessToken, stored);
  }
}

describe("sceneCreate", () => {
  it("places a scene first, or after the scene it names", async () => {
    const second = await sceneCreate(api, accessToken, { projectID: project.id, title: "Second", color: COLOR });
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First", color: COLOR });
    const third = await sceneCreate(api, accessToken, {
      projectID: project.id,
      after: second.id,
      title: "Third",
      color: COLOR,
    });

    expect(first.position).toBe(0);
    expect(third.position).toBe(2);
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Third"]);

    await deleteScenes(first, second, third);
  });

  it("returns 400 for an anchor outside the project", async () => {
    await expectStatus(
      sceneCreate(api, accessToken, { projectID: project.id, after: ABSENT_ID, title: "Scene", color: COLOR }),
      400
    );
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneCreate(api, accessToken, { projectID: ABSENT_ID, title: "Scene", color: COLOR }), 404);
  });
});

describe("sceneUpdate", () => {
  it("replaces the fields", async () => {
    const created = await sceneCreate(api, accessToken, { projectID: project.id, title: "Draft", color: COLOR });

    const scene = await sceneUpdate(api, accessToken, {
      projectID: project.id,
      id: created.id,
      title: "Ouverture",
      color: "#E0457B",
      description: "Marie attend.",
      prompt: "Lent.",
    });
    expect(scene).toMatchObject({
      title: "Ouverture",
      color: "#e0457b",
      description: "Marie attend.",
      prompt: "Lent.",
    });

    await deleteScenes(scene);
  });

  it("returns 404 for a scene of another project", async () => {
    const other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });
    const created = await sceneCreate(api, accessToken, { projectID: project.id, title: "Draft", color: COLOR });

    const request = { projectID: other.id, id: created.id, title: "Renamed", color: COLOR };
    await expectStatus(sceneUpdate(api, accessToken, request), 404);

    await deleteScenes(created);
    await projectDelete(api, accessToken, other);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(
      sceneUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, title: "Renamed", color: COLOR }),
      404
    );
  });
});

describe("sceneMove", () => {
  it("moves a scene", async () => {
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First", color: COLOR });
    const second = await sceneCreate(api, accessToken, {
      projectID: project.id,
      after: first.id,
      title: "Second",
      color: COLOR,
    });

    const moved = await sceneMove(api, accessToken, { projectID: project.id, id: first.id, after: second.id });
    expect(moved).toMatchObject({ id: first.id, position: 1 });
    await expect(sceneTitles()).resolves.toEqual(["Second", "First"]);

    const back = await sceneMove(api, accessToken, { projectID: project.id, id: first.id });
    expect(back).toMatchObject({ id: first.id, position: 0 });
    await expect(sceneTitles()).resolves.toEqual(["First", "Second"]);

    await deleteScenes(first, second);
  });

  it("keeps positions contiguous under concurrent moves", async () => {
    const scenes: Scene[] = [];
    for (let i = 0; i < 6; i++) {
      scenes.push(
        await sceneCreate(api, accessToken, {
          projectID: project.id,
          after: scenes.at(-1)?.id,
          title: "Scene",
          color: COLOR,
        })
      );
    }

    await Promise.all(scenes.map((scene) => sceneMove(api, accessToken, { projectID: project.id, id: scene.id })));

    const stored = await storedScenes();
    expect(stored.map((scene) => scene.position)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(stored.map((scene) => scene.id))).toEqual(new Set(scenes.map((scene) => scene.id)));

    await deleteScenes(...scenes);
  });

  it("returns 400 for a scene placed after itself", async () => {
    const scene = await sceneCreate(api, accessToken, { projectID: project.id, title: "Scene", color: COLOR });

    await expectStatus(sceneMove(api, accessToken, { projectID: project.id, id: scene.id, after: scene.id }), 400);

    await deleteScenes(scene);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneMove(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
  });
});

describe("sceneDelete", () => {
  it("deletes the scene and closes the gap", async () => {
    const first = await sceneCreate(api, accessToken, { projectID: project.id, title: "First", color: COLOR });
    const second = await sceneCreate(api, accessToken, {
      projectID: project.id,
      after: first.id,
      title: "Second",
      color: COLOR,
    });

    await expect(sceneDelete(api, accessToken, first)).resolves.toMatchObject({ id: first.id });

    const scenes = await storedScenes();
    expect(scenes.map((scene) => [scene.position, scene.title])).toEqual([[0, "Second"]]);

    await deleteScenes(second);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneDelete(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
  });
});

describe("sceneList", () => {
  it("pages through the scenes in narrative order, with their total", async () => {
    const scenes: Scene[] = [];
    for (const title of ["One", "Two", "Three"]) {
      scenes.push(
        await sceneCreate(api, accessToken, { projectID: project.id, after: scenes.at(-1)?.id, title, color: COLOR })
      );
    }

    const first = await sceneList(api, accessToken, { projectID: project.id, limit: 2 });
    expect(first.total).toBe(3);
    expect(first.scenes.map((scene) => scene.title)).toEqual(["One", "Two"]);

    const second = await sceneList(api, accessToken, { projectID: project.id, limit: 2, offset: 2 });
    expect(second.total).toBe(3);
    expect(second.scenes.map((scene) => scene.title)).toEqual(["Three"]);

    await expect(sceneList(api, accessToken, { projectID: project.id, offset: 3 })).resolves.toEqual({
      scenes: [],
      total: 0,
    });

    await deleteScenes(...scenes);
  });

  it("returns 400 for a page over 500 scenes", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
