import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  type Layer,
  LayerKind,
  NarrativeEngineApi,
  type ProjectComposer,
  layerCreate,
  layerDelete,
  layerList,
  layerMove,
  layerUpdate,
  layersCreate,
  layersDelete,
  layersUpdate,
  projectCreate,
  projectDelete,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

let project: ProjectComposer;
let manuscript: Layer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: "Layers", language: Language.En });
  manuscript = project.layers[0];
});

afterAll(async () => {
  await projectDelete(api, accessToken, project);
});

/** Adds one notes layer to the test project, at position or first. */
async function createNotes(name: string, position?: number): Promise<Layer> {
  return await layerCreate(api, accessToken, { projectID: project.id, position, kind: LayerKind.Notes, name });
}

/** Returns every layer of the test project, in order. */
async function storedLayers(): Promise<Layer[]> {
  return (await layerList(api, accessToken, { projectID: project.id, limit: 500 })).layers;
}

/** Returns the layer names of the test project, in order. */
async function layerNames(): Promise<string[]> {
  return (await storedLayers()).map((layer) => layer.name);
}

/** Deletes every layer of the test project but its first manuscript. */
async function deleteLayers(): Promise<void> {
  const ids = (await storedLayers()).filter((layer) => layer.id !== manuscript.id).map((layer) => layer.id);
  if (ids.length > 0) {
    await layersDelete(api, accessToken, { projectID: project.id, ids });
  }
}

describe("layerCreate", () => {
  it("places a layer at its position, and appends one past the end", async () => {
    const characters = await createNotes("Characters", 99);
    const places = await createNotes("Places");

    expect(characters).toMatchObject({ kind: LayerKind.Notes, position: 1 });
    expect(places.position).toBe(0);
    await expect(layerNames()).resolves.toEqual(["Places", manuscript.name, "Characters"]);

    await deleteLayers();
  });

  it("names the invalid field of a blank name", async () => {
    await expect(createNotes(" ")).rejects.toThrow('"notblank"');
  });

  it("returns 400 for a kind with no content schema", async () => {
    await expectStatus(
      layerCreate(api, accessToken, { projectID: project.id, kind: "chapter", name: "Chapters" }),
      400
    );
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(
      layerCreate(api, accessToken, { projectID: ABSENT_ID, kind: LayerKind.Notes, name: "Characters" }),
      404
    );
  });
});

describe("layersCreate", () => {
  it("places layers of any kind at their position, in order", async () => {
    const [characters, draft] = await layersCreate(api, accessToken, {
      projectID: project.id,
      position: 99,
      layers: [
        { kind: LayerKind.Notes, name: "Characters", description: "Who appears" },
        { kind: LayerKind.Manuscript, name: "Draft" },
      ],
    });

    expect(characters).toMatchObject({ kind: LayerKind.Notes, position: 1, description: "Who appears" });
    expect(draft).toMatchObject({ kind: LayerKind.Manuscript, position: 2 });
    await expect(layerNames()).resolves.toEqual([manuscript.name, "Characters", "Draft"]);

    await deleteLayers();
  });

  it("returns 400 for an empty list", async () => {
    await expectStatus(layersCreate(api, accessToken, { projectID: project.id, layers: [] }), 400);
  });
});

describe("layerUpdate", () => {
  it("replaces the fields", async () => {
    const created = await createNotes("Draft", 1);

    const layer = await layerUpdate(api, accessToken, {
      projectID: project.id,
      id: created.id,
      name: "Characters",
      description: "Who appears",
    });
    expect(layer).toMatchObject({ name: "Characters", description: "Who appears" });

    await deleteLayers();
  });

  it("returns 404 for an absent layer", async () => {
    await expectStatus(layerUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, name: "Renamed" }), 404);
  });
});

describe("layersUpdate", () => {
  it("replaces the fields of several layers, in the order of the request", async () => {
    const notes = await createNotes("Notes", 1);

    const updated = await layersUpdate(api, accessToken, {
      projectID: project.id,
      layers: [
        { id: notes.id, name: "Characters", description: "Who appears" },
        { id: manuscript.id, name: "Text" },
      ],
    });

    expect(updated.map((layer) => [layer.id, layer.name])).toEqual([
      [notes.id, "Characters"],
      [manuscript.id, "Text"],
    ]);
    await expect(layerNames()).resolves.toEqual(["Text", "Characters"]);

    await layerUpdate(api, accessToken, { projectID: project.id, id: manuscript.id, name: manuscript.name });
    await deleteLayers();
  });

  it("updates none of the layers when one is absent", async () => {
    const notes = await createNotes("Kept", 1);

    const layers = [
      { id: notes.id, name: "Renamed" },
      { id: ABSENT_ID, name: "Absent" },
    ];
    await expectStatus(layersUpdate(api, accessToken, { projectID: project.id, layers }), 404);
    await expect(layerNames()).resolves.toEqual([manuscript.name, "Kept"]);

    await deleteLayers();
  });
});

describe("layerMove", () => {
  it("moves a layer", async () => {
    const notes = await createNotes("Notes", 1);

    const moved = await layerMove(api, accessToken, { projectID: project.id, id: notes.id, position: 0 });
    expect(moved).toMatchObject({ id: notes.id, position: 0 });
    await expect(layerNames()).resolves.toEqual(["Notes", manuscript.name]);

    const last = await layerMove(api, accessToken, { projectID: project.id, id: notes.id, position: 99 });
    expect(last).toMatchObject({ id: notes.id, position: 1 });
    await expect(layerNames()).resolves.toEqual([manuscript.name, "Notes"]);

    await deleteLayers();
  });

  it("returns 404 for an absent layer", async () => {
    await expectStatus(layerMove(api, accessToken, { projectID: project.id, id: ABSENT_ID, position: 0 }), 404);
  });
});

describe("layerDelete", () => {
  it("deletes a layer and closes the gap", async () => {
    const notes = await createNotes("Notes");

    await expect(layerDelete(api, accessToken, { projectID: project.id, id: notes.id })).resolves.toMatchObject({
      id: notes.id,
    });
    await expect(layerNames()).resolves.toEqual([manuscript.name]);
  });

  it("deletes a manuscript layer while another remains", async () => {
    const draft = await layerCreate(api, accessToken, {
      projectID: project.id,
      position: 1,
      kind: LayerKind.Manuscript,
      name: "Draft",
    });

    await expect(layerDelete(api, accessToken, { projectID: project.id, id: draft.id })).resolves.toMatchObject({
      id: draft.id,
    });
  });

  it("returns 422 for the last layer of a required kind", async () => {
    await expectStatus(layerDelete(api, accessToken, { projectID: project.id, id: manuscript.id }), 422);
  });
});

describe("layersDelete", () => {
  it("deletes layers and closes the gaps", async () => {
    const [first, , third] = await layersCreate(api, accessToken, {
      projectID: project.id,
      position: 1,
      layers: ["One", "Two", "Three"].map((name) => ({ kind: LayerKind.Notes, name })),
    });

    const deleted = await layersDelete(api, accessToken, { projectID: project.id, ids: [first.id, third.id] });
    expect(deleted.map((layer) => layer.name)).toEqual(["One", "Three"]);
    await expect(layerNames()).resolves.toEqual([manuscript.name, "Two"]);

    await deleteLayers();
  });

  it("returns 422 when the delete would leave no layer of a required kind", async () => {
    await expectStatus(layersDelete(api, accessToken, { projectID: project.id, ids: [manuscript.id] }), 422);
  });

  it("deletes none of the layers when one is absent", async () => {
    const notes = await createNotes("Kept", 1);

    await expectStatus(layersDelete(api, accessToken, { projectID: project.id, ids: [notes.id, ABSENT_ID] }), 404);
    await expect(layerNames()).resolves.toEqual([manuscript.name, "Kept"]);

    await deleteLayers();
  });
});

describe("layerList", () => {
  it("pages through the layers in order, with their total", async () => {
    const notes = await createNotes("Notes", 1);

    const first = await layerList(api, accessToken, { projectID: project.id, limit: 1 });
    expect(first).toEqual({ layers: [expect.objectContaining({ id: manuscript.id })], total: 2 });

    const second = await layerList(api, accessToken, { projectID: project.id, limit: 1, offset: 1 });
    expect(second).toEqual({ layers: [expect.objectContaining({ id: notes.id })], total: 2 });

    await deleteLayers();
  });

  it("returns 400 for a page over 500 layers", async () => {
    await expectStatus(layerList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(layerList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
