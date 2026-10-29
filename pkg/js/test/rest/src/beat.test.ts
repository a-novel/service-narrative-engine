import { superAdminAccessToken } from "./token";

import { afterAll, afterEach, beforeAll, describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type Beat,
  Language,
  NarrativeEngineApi,
  type ProjectComposer,
  beatCreate,
  beatDelete,
  beatList,
  beatMove,
  beatUpdate,
  beatsCreate,
  beatsDelete,
  beatsUpdate,
  cellList,
  cellWrite,
  projectCreate,
  projectDelete,
  sceneList,
  scenesCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const COLOR = "#4a90d9";

let project: ProjectComposer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: "Beats", language: Language.En });
});

afterAll(async () => {
  await projectDelete(api, accessToken, project);
});

/** Adds one beat to the test project, at position or first. */
async function createBeat(title: string, position?: number): Promise<Beat> {
  return await beatCreate(api, accessToken, { projectID: project.id, position, title, color: COLOR });
}

/** Returns every beat of the test project, in narrative order. */
async function storedBeats(): Promise<Beat[]> {
  return (await beatList(api, accessToken, { projectID: project.id, limit: 500 })).beats;
}

/** Returns the beat titles of the test project, in narrative order. */
async function beatTitles(): Promise<string[]> {
  return (await storedBeats()).map((beat) => beat.title);
}

/** Deletes every beat of the test project, a batch of 100 at a time. */
async function deleteBeats(): Promise<void> {
  const ids = (await storedBeats()).map((beat) => beat.id);
  for (let start = 0; start < ids.length; start += 100) {
    await beatsDelete(api, accessToken, { projectID: project.id, ids: ids.slice(start, start + 100) });
  }
}

/** Fills the test project up to the 200 beats it may hold. */
async function fillBeats(): Promise<void> {
  const count = (await storedBeats()).length;
  for (let remaining = 200 - count; remaining > 0; remaining -= 100) {
    const beats = Array.from({ length: Math.min(100, remaining) }, () => ({ title: "Filler", color: COLOR }));
    await beatsCreate(api, accessToken, { projectID: project.id, position: 200, beats });
  }
}

afterEach(deleteBeats);

describe("beatCreate", () => {
  it("places a beat at its position, and appends one past the end", async () => {
    const second = await createBeat("Second");
    const first = await createBeat("First");
    const last = await createBeat("Last", 99);

    expect(second.position).toBe(0);
    expect(first.position).toBe(0);
    expect(last.position).toBe(2);
    await expect(beatTitles()).resolves.toEqual(["First", "Second", "Last"]);
  });

  it("returns 400 for a negative position", async () => {
    await expectStatus(createBeat("Beat", -1), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(beatCreate(api, accessToken, { projectID: ABSENT_ID, title: "Beat", color: COLOR }), 404);
  });

  it("returns 422 when the project holds 200 beats", async () => {
    await fillBeats();

    await expectStatus(createBeat("Extra"), 422);
  });
});

describe("beatsCreate", () => {
  it("places beats at their position, in order", async () => {
    await createBeat("Last");
    const [first, second] = await beatsCreate(api, accessToken, {
      projectID: project.id,
      beats: [
        { title: "First", color: COLOR },
        { title: "Second", color: "#E0457B", description: "Marie attend.", prompt: "Lent." },
      ],
    });

    expect(first).toMatchObject({ position: 0, title: "First" });
    expect(second).toMatchObject({ position: 1, color: "#e0457b", description: "Marie attend.", prompt: "Lent." });
    await expect(beatTitles()).resolves.toEqual(["First", "Second", "Last"]);
  });

  it("inserts the batch at its position", async () => {
    await createBeat("First");
    await createBeat("Last", 1);

    await beatsCreate(api, accessToken, {
      projectID: project.id,
      position: 1,
      beats: ["Two", "Three"].map((title) => ({ title, color: COLOR })),
    });

    await expect(beatTitles()).resolves.toEqual(["First", "Two", "Three", "Last"]);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(
      beatsCreate(api, accessToken, { projectID: ABSENT_ID, beats: [{ title: "Beat", color: COLOR }] }),
      404
    );
  });

  it("returns 422 when the batch would pass 200 beats", async () => {
    await fillBeats();

    await expectStatus(
      beatsCreate(api, accessToken, { projectID: project.id, beats: [{ title: "Extra", color: COLOR }] }),
      422
    );
  });

  it("returns 400 for an empty list or more than 100 beats", async () => {
    const beat = { title: "Beat", color: COLOR };

    await expectStatus(beatsCreate(api, accessToken, { projectID: project.id, beats: [] }), 400);
    await expectStatus(
      beatsCreate(api, accessToken, { projectID: project.id, beats: Array.from({ length: 101 }, () => beat) }),
      400
    );
  });

  it("adds none of the beats when one is invalid", async () => {
    const beats = [
      { title: "Valid", color: COLOR },
      { title: " ", color: COLOR },
    ];

    await expectStatus(beatsCreate(api, accessToken, { projectID: project.id, beats }), 400);
    await expect(storedBeats()).resolves.toEqual([]);
  });
});

describe("beatUpdate", () => {
  it("replaces the fields", async () => {
    const created = await createBeat("Draft");

    const beat = await beatUpdate(api, accessToken, {
      projectID: project.id,
      id: created.id,
      title: "Ouverture",
      color: "#E0457B",
      description: "Marie attend.",
      prompt: "Lent.",
    });
    expect(beat).toMatchObject({
      title: "Ouverture",
      color: "#e0457b",
      description: "Marie attend.",
      prompt: "Lent.",
    });
  });

  it("returns 404 for a beat of another project", async () => {
    const other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });
    onTestFinished(() => projectDelete(api, accessToken, other).then(() => undefined));
    const created = await createBeat("Draft");

    const request = { projectID: other.id, id: created.id, title: "Renamed", color: COLOR };
    await expectStatus(beatUpdate(api, accessToken, request), 404);
  });

  it("returns 404 for an absent beat", async () => {
    await expectStatus(
      beatUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, title: "Renamed", color: COLOR }),
      404
    );
  });
});

describe("beatsUpdate", () => {
  it("replaces the fields of several beats, in the order of the request", async () => {
    const [first, second] = await beatsCreate(api, accessToken, {
      projectID: project.id,
      beats: ["One", "Two"].map((title) => ({ title, color: COLOR })),
    });

    const updated = await beatsUpdate(api, accessToken, {
      projectID: project.id,
      beats: [
        { id: second.id, title: "Deux", color: "#E0457B" },
        { id: first.id, title: "Un", color: COLOR, description: "Marie attend." },
      ],
    });

    expect(updated.map((beat) => [beat.id, beat.title, beat.color])).toEqual([
      [second.id, "Deux", "#e0457b"],
      [first.id, "Un", COLOR],
    ]);
    await expect(beatTitles()).resolves.toEqual(["Un", "Deux"]);
  });

  it("updates none of the beats when one is absent", async () => {
    const beat = await createBeat("Kept");

    const beats = [
      { id: beat.id, title: "Renamed", color: COLOR },
      { id: ABSENT_ID, title: "Absent", color: COLOR },
    ];
    await expectStatus(beatsUpdate(api, accessToken, { projectID: project.id, beats }), 404);
    await expect(beatTitles()).resolves.toEqual(["Kept"]);
  });

  it("returns 400 for a beat named twice", async () => {
    const beat = await createBeat("Twice");

    const item = { id: beat.id, title: "Renamed", color: COLOR };
    await expectStatus(beatsUpdate(api, accessToken, { projectID: project.id, beats: [item, item] }), 400);
  });
});

describe("beatMove", () => {
  it("moves a beat", async () => {
    const first = await createBeat("First");
    await createBeat("Second", 1);

    const moved = await beatMove(api, accessToken, { projectID: project.id, id: first.id, position: 1 });
    expect(moved).toMatchObject({ id: first.id, position: 1 });
    await expect(beatTitles()).resolves.toEqual(["Second", "First"]);

    const back = await beatMove(api, accessToken, { projectID: project.id, id: first.id, position: 0 });
    expect(back).toMatchObject({ id: first.id, position: 0 });
    await expect(beatTitles()).resolves.toEqual(["First", "Second"]);

    const last = await beatMove(api, accessToken, { projectID: project.id, id: first.id, position: 99 });
    expect(last).toMatchObject({ id: first.id, position: 1 });
    await expect(beatTitles()).resolves.toEqual(["Second", "First"]);
  });

  it("keeps positions contiguous under concurrent moves", async () => {
    const beats = await beatsCreate(api, accessToken, {
      projectID: project.id,
      beats: Array.from({ length: 6 }, () => ({ title: "Beat", color: COLOR })),
    });

    await Promise.all(
      beats.map((beat) => beatMove(api, accessToken, { projectID: project.id, id: beat.id, position: 0 }))
    );

    const stored = await storedBeats();
    expect(stored.map((beat) => beat.position)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(stored.map((beat) => beat.id))).toEqual(new Set(beats.map((beat) => beat.id)));
  });

  it("returns 400 for a negative position", async () => {
    const beat = await createBeat("Beat");

    await expectStatus(beatMove(api, accessToken, { projectID: project.id, id: beat.id, position: -1 }), 400);
  });

  it("returns 404 for an absent beat", async () => {
    await expectStatus(beatMove(api, accessToken, { projectID: project.id, id: ABSENT_ID, position: 0 }), 404);
  });
});

describe("beatDelete", () => {
  it("deletes a beat and closes the gap", async () => {
    const first = await createBeat("First");
    await createBeat("Second", 1);

    await expect(beatDelete(api, accessToken, { projectID: project.id, id: first.id })).resolves.toMatchObject({
      id: first.id,
    });
    await expect(storedBeats()).resolves.toEqual([expect.objectContaining({ position: 0, title: "Second" })]);
  });

  it("deletes the beat's scenes and their cells", async () => {
    const beat = await createBeat("Doomed");
    const [scene] = await scenesCreate(api, accessToken, {
      projectID: project.id,
      beatID: beat.id,
      scenes: [{ title: "La fenêtre" }],
    });
    const content = { blocks: [{ kind: "narration" as const, text: "Marie attend." }] };
    await cellWrite(api, accessToken, {
      projectID: project.id,
      sceneID: scene.id,
      layerID: project.layers[0].id,
      content,
    });

    await beatDelete(api, accessToken, { projectID: project.id, id: beat.id });

    await expect(sceneList(api, accessToken, { projectID: project.id })).resolves.toEqual({ scenes: [], total: 0 });
    await expect(cellList(api, accessToken, { projectID: project.id })).resolves.toEqual({ cells: [], total: 0 });
  });

  it("returns 404 for an absent beat", async () => {
    await expectStatus(beatDelete(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
  });
});

describe("beatsDelete", () => {
  it("deletes beats and closes the gaps", async () => {
    const beats = await beatsCreate(api, accessToken, {
      projectID: project.id,
      beats: ["One", "Two", "Three", "Four"].map((title) => ({ title, color: COLOR })),
    });

    const deleted = await beatsDelete(api, accessToken, { projectID: project.id, ids: [beats[0].id, beats[2].id] });
    expect(deleted.map((beat) => beat.title)).toEqual(["One", "Three"]);

    const stored = await storedBeats();
    expect(stored.map((beat) => [beat.position, beat.title])).toEqual([
      [0, "Two"],
      [1, "Four"],
    ]);
  });

  it("deletes none of the beats when one is absent", async () => {
    const beat = await createBeat("Kept");

    await expectStatus(beatsDelete(api, accessToken, { projectID: project.id, ids: [beat.id, ABSENT_ID] }), 404);
    await expect(beatTitles()).resolves.toEqual(["Kept"]);
  });

  it("returns 400 for an empty list", async () => {
    await expectStatus(beatsDelete(api, accessToken, { projectID: project.id, ids: [] }), 400);
  });
});

describe("beatList", () => {
  it("pages through the beats in narrative order, with their total", async () => {
    await beatsCreate(api, accessToken, {
      projectID: project.id,
      beats: ["One", "Two", "Three"].map((title) => ({ title, color: COLOR })),
    });

    const first = await beatList(api, accessToken, { projectID: project.id, limit: 2 });
    expect(first.total).toBe(3);
    expect(first.beats.map((beat) => beat.title)).toEqual(["One", "Two"]);

    const second = await beatList(api, accessToken, { projectID: project.id, limit: 2, offset: 2 });
    expect(second.total).toBe(3);
    expect(second.beats.map((beat) => beat.title)).toEqual(["Three"]);

    await expect(beatList(api, accessToken, { projectID: project.id, offset: 3 })).resolves.toEqual({
      beats: [],
      total: 0,
    });
  });

  it("returns 400 for a page over 500 beats", async () => {
    await expectStatus(beatList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(beatList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
