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
  layerMove,
  layerUpdate,
  projectCreate,
  projectDelete,
  projectGet,
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
  await projectDelete(api, accessToken, await projectGet(api, accessToken, project));
});

/** Returns the layer names of the test project, in order. */
async function layerNames(): Promise<string[]> {
  return (await projectGet(api, accessToken, project)).layers.map((layer) => layer.name);
}

/** Deletes every layer the test created. */
async function deleteLayers(...layers: Layer[]): Promise<void> {
  for (const layer of layers) {
    const stored = (await projectGet(api, accessToken, project)).layers.find((item) => item.id === layer.id);
    if (stored) await layerDelete(api, accessToken, stored);
  }
}

describe("layerCreate", () => {
  it("places a notes layer first, or after the layer it names", async () => {
    const characters = await layerCreate(api, accessToken, {
      projectID: project.id,
      after: manuscript.id,
      name: "Characters",
      description: "Who appears",
    });
    const places = await layerCreate(api, accessToken, { projectID: project.id, name: "Places" });

    expect(characters).toMatchObject({ kind: LayerKind.Notes, position: 1, description: "Who appears" });
    expect(places.position).toBe(0);
    await expect(layerNames()).resolves.toEqual(["Places", manuscript.name, "Characters"]);

    await deleteLayers(characters, places);
  });

  it("names the invalid field of a blank name", async () => {
    await expect(layerCreate(api, accessToken, { projectID: project.id, name: " " })).rejects.toThrow(
      '"invalidFields":{"name":"notblank"}'
    );
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(layerCreate(api, accessToken, { projectID: ABSENT_ID, name: "Characters" }), 404);
  });
});

describe("layerUpdate", () => {
  it("replaces the fields", async () => {
    const created = await layerCreate(api, accessToken, { projectID: project.id, after: manuscript.id, name: "Draft" });

    const layer = await layerUpdate(api, accessToken, {
      id: created.id,
      name: "Characters",
      description: "Who appears",
    });
    expect(layer).toMatchObject({ name: "Characters", description: "Who appears" });

    await deleteLayers(layer);
  });

  it("returns 404 for an absent layer", async () => {
    await expectStatus(layerUpdate(api, accessToken, { id: ABSENT_ID, name: "Renamed" }), 404);
  });
});

describe("layerMove", () => {
  it("moves a layer", async () => {
    const notes = await layerCreate(api, accessToken, { projectID: project.id, after: manuscript.id, name: "Notes" });

    const layers = await layerMove(api, accessToken, { id: notes.id });
    expect(layers.map((layer) => [layer.position, layer.name])).toEqual([
      [0, "Notes"],
      [1, manuscript.name],
    ]);

    await deleteLayers(notes);
  });

  it("returns 404 for an absent layer", async () => {
    await expectStatus(layerMove(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("layerDelete", () => {
  it("deletes a notes layer and closes the gap", async () => {
    const notes = await layerCreate(api, accessToken, { projectID: project.id, name: "Notes" });

    await expect(layerDelete(api, accessToken, notes)).resolves.toMatchObject({ id: notes.id });

    const layers = (await projectGet(api, accessToken, project)).layers;
    expect(layers.map((layer) => [layer.position, layer.kind])).toEqual([[0, LayerKind.Manuscript]]);
  });

  it("returns 422 for the manuscript layer", async () => {
    await expectStatus(layerDelete(api, accessToken, manuscript), 422);
  });
});
