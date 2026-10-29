import { superAdminAccessToken } from "./token";

import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type Beat,
  Language,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  beatCreate,
  cellList,
  cellWrite,
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
// The test project's beats: scenes go to the first unless a test names the second.
let first: Beat;
let second: Beat;
let other: ProjectComposer;
let otherBeat: Beat;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: "Scenes", language: Language.En });
  first = await beatCreate(api, accessToken, { projectID: project.id, title: "First beat", color: COLOR });
  second = await beatCreate(api, accessToken, {
    projectID: project.id,
    position: 1,
    title: "Second beat",
    color: COLOR,
  });
  other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });
  otherBeat = await beatCreate(api, accessToken, { projectID: other.id, title: "Other beat", color: COLOR });
});

afterAll(async () => {
  for (const item of [project, other]) {
    await projectDelete(api, accessToken, item);
  }
});

/** Adds one scene to a beat of the test project, the first unless named, at position or first. */
async function createScene(title: string, position?: number, beat: Beat = first): Promise<Scene> {
  return await sceneCreate(api, accessToken, { projectID: project.id, beatID: beat.id, position, title });
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
  const ids = (await storedScenes()).map((scene) => scene.id);
  if (ids.length) await scenesDelete(api, accessToken, { projectID: project.id, ids });
}

/** Fills the first beat up to the 10 scenes it may hold. */
async function fillBeat(): Promise<void> {
  const count = (await storedScenes()).filter((scene) => scene.beatID === first.id).length;
  const scenes = Array.from({ length: 10 - count }, () => ({ title: "Filler" }));
  await scenesCreate(api, accessToken, { projectID: project.id, beatID: first.id, position: 10, scenes });
}

afterEach(deleteScenes);

describe("sceneCreate", () => {
  it("places a scene at its position in its beat, and appends one past the end", async () => {
    const secondScene = await createScene("Second");
    const firstScene = await createScene("First");
    const last = await createScene("Last", 99);

    expect(secondScene).toMatchObject({ beatID: first.id, position: 0 });
    expect(firstScene.position).toBe(0);
    expect(last.position).toBe(2);
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Last"]);
  });

  it("returns 400 for a negative position", async () => {
    await expectStatus(createScene("Scene", -1), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneCreate(api, accessToken, { projectID: ABSENT_ID, beatID: first.id, title: "Scene" }), 404);
  });

  it("returns 404 for a beat of another project", async () => {
    await expectStatus(createScene("Scene", 0, otherBeat), 404);
  });

  it("returns 422 when the beat holds 10 scenes", async () => {
    await fillBeat();

    await expectStatus(createScene("Extra"), 422);
  });
});

describe("scenesCreate", () => {
  it("places scenes at their position in their beat, in order", async () => {
    await createScene("Last");
    const [one, two] = await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      scenes: [{ title: "First" }, { title: "Second", description: "Marie attend.", prompt: "Lent." }],
    });

    expect(one).toMatchObject({ beatID: first.id, position: 0, title: "First", description: "" });
    expect(two).toMatchObject({ position: 1, description: "Marie attend.", prompt: "Lent." });
    await expect(sceneTitles()).resolves.toEqual(["First", "Second", "Last"]);
  });

  it("inserts the batch at its position", async () => {
    await createScene("First");
    await createScene("Last", 1);

    await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      position: 1,
      scenes: [{ title: "Two" }, { title: "Three" }],
    });

    await expect(sceneTitles()).resolves.toEqual(["First", "Two", "Three", "Last"]);
  });

  it("returns 404 for an absent project", async () => {
    const request = { projectID: ABSENT_ID, beatID: first.id, scenes: [{ title: "Scene" }] };
    await expectStatus(scenesCreate(api, accessToken, request), 404);
  });

  it("returns 422 when the batch would pass 10 scenes in the beat", async () => {
    await fillBeat();

    const request = { projectID: project.id, beatID: first.id, scenes: [{ title: "Extra" }] };
    await expectStatus(scenesCreate(api, accessToken, request), 422);
  });

  it("returns 400 for an empty list or more than 100 scenes", async () => {
    const scenes = Array.from({ length: 101 }, () => ({ title: "Scene" }));

    await expectStatus(scenesCreate(api, accessToken, { projectID: project.id, beatID: first.id, scenes: [] }), 400);
    await expectStatus(scenesCreate(api, accessToken, { projectID: project.id, beatID: first.id, scenes }), 400);
  });

  it("adds none of the scenes when one is invalid", async () => {
    const scenes = [{ title: "Valid" }, { title: " " }];

    await expectStatus(scenesCreate(api, accessToken, { projectID: project.id, beatID: first.id, scenes }), 400);
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
      description: "Marie attend.",
      prompt: "Lent.",
    });
    expect(scene).toMatchObject({ title: "Ouverture", description: "Marie attend.", prompt: "Lent." });
  });

  it("returns 404 for a scene of another project", async () => {
    const created = await createScene("Draft");

    await expectStatus(sceneUpdate(api, accessToken, { projectID: other.id, id: created.id, title: "Renamed" }), 404);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, title: "Renamed" }), 404);
  });
});

describe("scenesUpdate", () => {
  it("replaces the fields of several scenes, in the order of the request", async () => {
    const [one, two] = await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      scenes: [{ title: "One" }, { title: "Two" }],
    });

    const updated = await scenesUpdate(api, accessToken, {
      projectID: project.id,
      scenes: [
        { id: two.id, title: "Deux" },
        { id: one.id, title: "Un", description: "Marie attend." },
      ],
    });

    expect(updated.map((scene) => [scene.id, scene.title, scene.description])).toEqual([
      [two.id, "Deux", ""],
      [one.id, "Un", "Marie attend."],
    ]);
    await expect(sceneTitles()).resolves.toEqual(["Un", "Deux"]);
  });

  it("updates none of the scenes when one is absent", async () => {
    const scene = await createScene("Kept");

    const scenes = [
      { id: scene.id, title: "Renamed" },
      { id: ABSENT_ID, title: "Absent" },
    ];
    await expectStatus(scenesUpdate(api, accessToken, { projectID: project.id, scenes }), 404);
    await expect(sceneTitles()).resolves.toEqual(["Kept"]);
  });

  it("returns 400 for a scene named twice", async () => {
    const scene = await createScene("Twice");

    const item = { id: scene.id, title: "Renamed" };
    await expectStatus(scenesUpdate(api, accessToken, { projectID: project.id, scenes: [item, item] }), 400);
  });
});

describe("sceneMove", () => {
  it("moves a scene within its beat", async () => {
    const scene = await createScene("First");
    await createScene("Second", 1);

    const moved = await sceneMove(api, accessToken, {
      projectID: project.id,
      id: scene.id,
      beatID: first.id,
      position: 1,
    });
    expect(moved).toMatchObject({ id: scene.id, position: 1 });
    await expect(sceneTitles()).resolves.toEqual(["Second", "First"]);

    const back = await sceneMove(api, accessToken, {
      projectID: project.id,
      id: scene.id,
      beatID: first.id,
      position: 0,
    });
    expect(back).toMatchObject({ id: scene.id, position: 0 });
    await expect(sceneTitles()).resolves.toEqual(["First", "Second"]);

    const last = await sceneMove(api, accessToken, {
      projectID: project.id,
      id: scene.id,
      beatID: first.id,
      position: 99,
    });
    expect(last).toMatchObject({ id: scene.id, position: 1 });
    await expect(sceneTitles()).resolves.toEqual(["Second", "First"]);
  });

  it("moves a scene to another beat, keeping its ID and its cells", async () => {
    const scene = await createScene("Moving");
    await createScene("Staying", 1);
    await createScene("Waiting", 0, second);
    const content = { blocks: [{ kind: "narration" as const, text: "Marie attend." }] };
    await cellWrite(api, accessToken, {
      projectID: project.id,
      sceneID: scene.id,
      layerID: project.layers[0].id,
      content,
    });

    const moved = await sceneMove(api, accessToken, {
      projectID: project.id,
      id: scene.id,
      beatID: second.id,
      position: 0,
    });
    expect(moved).toMatchObject({ id: scene.id, beatID: second.id, position: 0 });

    const stored = await storedScenes();
    expect(stored.map((item) => [item.title, item.beatID, item.position])).toEqual([
      ["Staying", first.id, 0],
      ["Moving", second.id, 0],
      ["Waiting", second.id, 1],
    ]);
    const { cells } = await cellList(api, accessToken, { projectID: project.id });
    expect(cells).toEqual([expect.objectContaining({ sceneID: scene.id, content })]);
  });

  it("keeps positions contiguous under concurrent moves", async () => {
    const scenes = await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      scenes: Array.from({ length: 6 }, () => ({ title: "Scene" })),
    });

    await Promise.all(
      scenes.map((scene) =>
        sceneMove(api, accessToken, { projectID: project.id, id: scene.id, beatID: first.id, position: 0 })
      )
    );

    const stored = await storedScenes();
    expect(stored.map((scene) => scene.position)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(stored.map((scene) => scene.id))).toEqual(new Set(scenes.map((scene) => scene.id)));
  });

  it("returns 400 for a negative position", async () => {
    const scene = await createScene("Scene");

    const request = { projectID: project.id, id: scene.id, beatID: first.id, position: -1 };
    await expectStatus(sceneMove(api, accessToken, request), 400);
  });

  it("returns 404 for an absent scene", async () => {
    const request = { projectID: project.id, id: ABSENT_ID, beatID: first.id, position: 0 };
    await expectStatus(sceneMove(api, accessToken, request), 404);
  });

  it("returns 404 for a beat of another project", async () => {
    const scene = await createScene("Scene");

    const request = { projectID: project.id, id: scene.id, beatID: otherBeat.id, position: 0 };
    await expectStatus(sceneMove(api, accessToken, request), 404);
  });

  it("returns 422 when the target beat holds 10 scenes", async () => {
    await fillBeat();
    const scene = await createScene("Extra", 0, second);

    const request = { projectID: project.id, id: scene.id, beatID: first.id, position: 0 };
    await expectStatus(sceneMove(api, accessToken, request), 422);
  });
});

describe("sceneDelete", () => {
  it("deletes a scene and closes the gap in its beat", async () => {
    const scene = await createScene("First");
    await createScene("Second", 1);

    await expect(sceneDelete(api, accessToken, { projectID: project.id, id: scene.id })).resolves.toMatchObject({
      id: scene.id,
    });
    await expect(storedScenes()).resolves.toEqual([expect.objectContaining({ position: 0, title: "Second" })]);
  });

  it("returns 404 for an absent scene", async () => {
    await expectStatus(sceneDelete(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
  });
});

describe("scenesDelete", () => {
  it("deletes scenes and closes the gaps", async () => {
    const scenes = await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      scenes: ["One", "Two", "Three", "Four"].map((title) => ({ title })),
    });

    const deleted = await scenesDelete(api, accessToken, { projectID: project.id, ids: [scenes[0].id, scenes[2].id] });
    expect(deleted.map((scene) => scene.title)).toEqual(["One", "Three"]);

    const stored = await storedScenes();
    expect(stored.map((scene) => [scene.position, scene.title])).toEqual([
      [0, "Two"],
      [1, "Four"],
    ]);
  });

  it("deletes none of the scenes when one is absent", async () => {
    const scene = await createScene("Kept");

    await expectStatus(scenesDelete(api, accessToken, { projectID: project.id, ids: [scene.id, ABSENT_ID] }), 404);
    await expect(sceneTitles()).resolves.toEqual(["Kept"]);
  });

  it("returns 400 for an empty list", async () => {
    await expectStatus(scenesDelete(api, accessToken, { projectID: project.id, ids: [] }), 400);
  });
});

describe("sceneList", () => {
  it("pages through the scenes in narrative order, with their total", async () => {
    await createScene("Three", 0, second);
    await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: first.id,
      scenes: [{ title: "One" }, { title: "Two" }],
    });

    const page = await sceneList(api, accessToken, { projectID: project.id, limit: 2 });
    expect(page.total).toBe(3);
    expect(page.scenes.map((scene) => scene.title)).toEqual(["One", "Two"]);

    const next = await sceneList(api, accessToken, { projectID: project.id, limit: 2, offset: 2 });
    expect(next.total).toBe(3);
    expect(next.scenes.map((scene) => scene.title)).toEqual(["Three"]);

    await expect(sceneList(api, accessToken, { projectID: project.id, offset: 3 })).resolves.toEqual({
      scenes: [],
      total: 0,
    });
  });

  it("returns 400 for a page over 500 scenes", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(sceneList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
