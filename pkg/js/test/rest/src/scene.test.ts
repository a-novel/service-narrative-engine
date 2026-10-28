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
  scenesCreate,
  scenesDelete,
  scenesUpdate,
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

/** Adds one scene to the test project, at position or first. */
async function createScene(title: string, position?: number): Promise<Scene> {
  return await sceneCreate(api, accessToken, { projectID: project.id, position, title, color: COLOR });
}

/** Returns every scene of the test project, in narrative order. */
async function storedScenes(): Promise<Scene[]> {
  return (await sceneList(api, accessToken, { projectID: project.id, limit: 500 })).scenes;
}

/** Returns the scene titles of the test project, in narrative order. */
async function sceneTitles(): Promise<string[]> {
  return (await storedScenes()).map((scene) => scene.title);
}

/** Deletes every scene of the test project. */
async function deleteScenes(): Promise<void> {
  const scenes = await storedScenes();
  if (scenes.length > 0) {
    await scenesDelete(api, accessToken, { projectID: project.id, ids: scenes.map((scene) => scene.id) });
  }
}

describe("sceneCreate", () => {
  it("places a scene at its position, and appends one past the end", async () => {
    const second = await createScene("Second");
    const first = await createScene("First");
    const last = await createScene("Last", 99);

    expect(second.position).toBe(0);
    expect(first.position).toBe(0);
    expect(last.position).toBe(2);
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Last"]);

    await deleteScenes();
  });

  it("returns 400 for a negative position", async () => {
    await expectStatus(createScene("Scene", -1), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneCreate(api, accessToken, { projectID: ABSENT_ID, title: "Scene", color: COLOR }), 404);
  });
});

describe("scenesCreate", () => {
  it("places scenes at their position, in order", async () => {
    await createScene("Last");
    const [first, second] = await scenesCreate(api, accessToken, {
      projectID: project.id,
      scenes: [
        { title: "First", color: COLOR },
        { title: "Second", color: "#E0457B", description: "Marie attend.", prompt: "Lent." },
      ],
    });

    expect(first).toMatchObject({ position: 0, title: "First" });
    expect(second).toMatchObject({ position: 1, color: "#e0457b", description: "Marie attend.", prompt: "Lent." });
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Last"]);

    await deleteScenes();
  });

  it("returns 400 for an empty list or more than 100 scenes", async () => {
    const scene = { title: "Scene", color: COLOR };

    await expectStatus(scenesCreate(api, accessToken, { projectID: project.id, scenes: [] }), 400);
    await expectStatus(
      scenesCreate(api, accessToken, { projectID: project.id, scenes: Array.from({ length: 101 }, () => scene) }),
      400
    );
  });

  it("adds none of the scenes when one is invalid", async () => {
    const scenes = [
      { title: "Valid", color: COLOR },
      { title: " ", color: COLOR },
    ];

    await expectStatus(scenesCreate(api, accessToken, { projectID: project.id, scenes }), 400);
    await expect(storedScenes()).resolves.toEqual([]);
  });
});

describe("sceneUpdate", () => {
  it("replaces the fields", async () => {
    const created = await createScene("Draft");

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

    await deleteScenes();
  });

  it("returns 404 for a scene of another project", async () => {
    const other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });
    const created = await createScene("Draft");

    const request = { projectID: other.id, id: created.id, title: "Renamed", color: COLOR };
    await expectStatus(sceneUpdate(api, accessToken, request), 404);

    await deleteScenes();
    await projectDelete(api, accessToken, other);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(
      sceneUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, title: "Renamed", color: COLOR }),
      404
    );
  });
});

describe("scenesUpdate", () => {
  it("replaces the fields of several scenes, in the order of the request", async () => {
    const [first, second] = await scenesCreate(api, accessToken, {
      projectID: project.id,
      scenes: ["One", "Two"].map((title) => ({ title, color: COLOR })),
    });

    const updated = await scenesUpdate(api, accessToken, {
      projectID: project.id,
      scenes: [
        { id: second.id, title: "Deux", color: "#E0457B" },
        { id: first.id, title: "Un", color: COLOR, description: "Marie attend." },
      ],
    });

    expect(updated.map((scene) => [scene.id, scene.title, scene.color])).toEqual([
      [second.id, "Deux", "#e0457b"],
      [first.id, "Un", COLOR],
    ]);
    await expect(sceneTitles()).resolves.toEqual(["Un", "Deux"]);

    await deleteScenes();
  });

  it("updates none of the scenes when one is absent", async () => {
    const scene = await createScene("Kept");

    const scenes = [
      { id: scene.id, title: "Renamed", color: COLOR },
      { id: ABSENT_ID, title: "Absent", color: COLOR },
    ];
    await expectStatus(scenesUpdate(api, accessToken, { projectID: project.id, scenes }), 404);
    await expect(sceneTitles()).resolves.toEqual(["Kept"]);

    await deleteScenes();
  });

  it("returns 400 for a scene named twice", async () => {
    const scene = await createScene("Twice");

    const item = { id: scene.id, title: "Renamed", color: COLOR };
    await expectStatus(scenesUpdate(api, accessToken, { projectID: project.id, scenes: [item, item] }), 400);

    await deleteScenes();
  });
});

describe("sceneMove", () => {
  it("moves a scene", async () => {
    const first = await createScene("First");
    await createScene("Second", 1);

    const moved = await sceneMove(api, accessToken, { projectID: project.id, id: first.id, position: 1 });
    expect(moved).toMatchObject({ id: first.id, position: 1 });
    await expect(sceneTitles()).resolves.toEqual(["Second", "First"]);

    const back = await sceneMove(api, accessToken, { projectID: project.id, id: first.id, position: 0 });
    expect(back).toMatchObject({ id: first.id, position: 0 });
    await expect(sceneTitles()).resolves.toEqual(["First", "Second"]);

    const last = await sceneMove(api, accessToken, { projectID: project.id, id: first.id, position: 99 });
    expect(last).toMatchObject({ id: first.id, position: 1 });
    await expect(sceneTitles()).resolves.toEqual(["Second", "First"]);

    await deleteScenes();
  });

  it("keeps positions contiguous under concurrent moves", async () => {
    const scenes = await scenesCreate(api, accessToken, {
      projectID: project.id,
      scenes: Array.from({ length: 6 }, () => ({ title: "Scene", color: COLOR })),
    });

    await Promise.all(
      scenes.map((scene) => sceneMove(api, accessToken, { projectID: project.id, id: scene.id, position: 0 }))
    );

    const stored = await storedScenes();
    expect(stored.map((scene) => scene.position)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(stored.map((scene) => scene.id))).toEqual(new Set(scenes.map((scene) => scene.id)));

    await deleteScenes();
  });

  it("returns 400 for a negative position", async () => {
    const scene = await createScene("Scene");

    await expectStatus(sceneMove(api, accessToken, { projectID: project.id, id: scene.id, position: -1 }), 400);

    await deleteScenes();
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneMove(api, accessToken, { projectID: project.id, id: ABSENT_ID, position: 0 }), 404);
  });
});

describe("sceneDelete", () => {
  it("deletes a scene and closes the gap", async () => {
    const first = await createScene("First");
    await createScene("Second", 1);

    await expect(sceneDelete(api, accessToken, { projectID: project.id, id: first.id })).resolves.toMatchObject({
      id: first.id,
    });
    await expect(storedScenes()).resolves.toEqual([expect.objectContaining({ position: 0, title: "Second" })]);

    await deleteScenes();
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneDelete(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
  });
});

describe("scenesDelete", () => {
  it("deletes scenes and closes the gaps", async () => {
    const scenes = await scenesCreate(api, accessToken, {
      projectID: project.id,
      scenes: ["One", "Two", "Three", "Four"].map((title) => ({ title, color: COLOR })),
    });

    const deleted = await scenesDelete(api, accessToken, { projectID: project.id, ids: [scenes[0].id, scenes[2].id] });
    expect(deleted.map((scene) => scene.title)).toEqual(["One", "Three"]);

    const stored = await storedScenes();
    expect(stored.map((scene) => [scene.position, scene.title])).toEqual([
      [0, "Two"],
      [1, "Four"],
    ]);

    await deleteScenes();
  });

  it("deletes none of the scenes when one is absent", async () => {
    const scene = await createScene("Kept");

    await expectStatus(scenesDelete(api, accessToken, { projectID: project.id, ids: [scene.id, ABSENT_ID] }), 404);
    await expect(sceneTitles()).resolves.toEqual(["Kept"]);

    await deleteScenes();
  });

  it("returns 400 for an empty list", async () => {
    await expectStatus(scenesDelete(api, accessToken, { projectID: project.id, ids: [] }), 400);
  });
});

describe("sceneList", () => {
  it("pages through the scenes in narrative order, with their total", async () => {
    await scenesCreate(api, accessToken, {
      projectID: project.id,
      scenes: ["One", "Two", "Three"].map((title) => ({ title, color: COLOR })),
    });

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

    await deleteScenes();
  });

  it("returns 400 for a page over 500 scenes", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
