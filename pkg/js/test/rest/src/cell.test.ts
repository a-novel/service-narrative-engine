import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  type Layer,
  LayerKind,
  ManuscriptContentSchema,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  cellList,
  cellWrite,
  layerCreate,
  projectCreate,
  projectDelete,
  sceneCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const COLOR = "#4a90d9";
const accessToken = await superAdminAccessToken();

let project: ProjectComposer;
let scenes: Scene[];
let other: ProjectComposer;
let otherScenes: Scene[];
let notes: Layer;

/** Creates a project with one scene per title. */
async function projectWithScenes(title: string, titles: string[]): Promise<[ProjectComposer, Scene[]]> {
  const created = await projectCreate(api, accessToken, { title, language: Language.En });
  const createdScenes = await sceneCreate(api, accessToken, {
    projectID: created.id,
    scenes: titles.map((sceneTitle) => ({ title: sceneTitle, color: COLOR })),
  });

  return [created, createdScenes];
}

beforeAll(async () => {
  [project, scenes] = await projectWithScenes("Cells", ["Opening", "Ending"]);
  [other, otherScenes] = await projectWithScenes("Other", ["Only"]);
  [notes] = await layerCreate(api, accessToken, {
    projectID: project.id,
    position: 1,
    layers: [{ kind: LayerKind.Notes, name: "Notes" }],
  });
});

afterAll(async () => {
  for (const item of [project, other]) {
    await projectDelete(api, accessToken, item);
  }
});

describe("cellWrite", () => {
  it("writes a manuscript cell, then rewrites it", async () => {
    const cell = {
      sceneID: scenes[0].id,
      layerID: project.layers[0].id,
      content: {
        blocks: [
          { kind: "narration" as const, text: "La pluie battait les vitres." },
          { kind: "dialogue" as const, speaker: "Marie", parts: [{ tone: "bas" }, { text: "Tu es venu." }] },
        ],
      },
    };

    const [written] = await cellWrite(api, accessToken, { projectID: project.id, cells: [cell] });
    expect(ManuscriptContentSchema.parse(written.content)).toEqual(cell.content);

    const [rewritten] = await cellWrite(api, accessToken, {
      projectID: project.id,
      cells: [{ ...cell, content: { blocks: [] } }],
    });
    expect(rewritten.content).toEqual({ blocks: [] });

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored).toContainEqual(rewritten);
  });

  it("writes cells of several layers at once, in the order of the request", async () => {
    const written = await cellWrite(api, accessToken, {
      projectID: project.id,
      cells: [
        { sceneID: scenes[1].id, layerID: notes.id, content: { text: "Marie" } },
        { sceneID: scenes[1].id, layerID: project.layers[0].id, content: { blocks: [] } },
      ],
    });

    expect(written.map((cell) => [cell.layerID, cell.content])).toEqual([
      [notes.id, { text: "Marie" }],
      [project.layers[0].id, { blocks: [] }],
    ]);
  });

  it("returns 400 for content outside the schema of the layer's kind, and writes none of the cells", async () => {
    const cells = [
      { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Kept out" } },
      { sceneID: scenes[0].id, layerID: project.layers[0].id, content: { text: "Not a manuscript" } },
    ];
    await expectStatus(cellWrite(api, accessToken, { projectID: project.id, cells }), 400);

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored.find((cell) => cell.sceneID === scenes[0].id && cell.layerID === notes.id)).toBeUndefined();
  });

  it("returns 400 for a scene and a layer named twice", async () => {
    const cell = { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Marie" } };
    await expectStatus(cellWrite(api, accessToken, { projectID: project.id, cells: [cell, cell] }), 400);
  });

  it("returns 404 for a scene of another project, and writes none of the cells", async () => {
    const cells = [
      { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Kept out" } },
      { sceneID: otherScenes[0].id, layerID: notes.id, content: { text: "Marie" } },
    ];
    await expectStatus(cellWrite(api, accessToken, { projectID: project.id, cells }), 404);

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored.find((cell) => cell.sceneID === scenes[0].id && cell.layerID === notes.id)).toBeUndefined();
  });

  it("returns 404 for a layer of another project", async () => {
    const cells = [{ sceneID: otherScenes[0].id, layerID: notes.id, content: { text: "Marie" } }];
    await expectStatus(cellWrite(api, accessToken, { projectID: other.id, cells }), 404);
  });
});

describe("cellList", () => {
  it("pages through the cells of a project, with their total", async () => {
    const [paged, pagedScenes] = await projectWithScenes("Paged", ["One", "Two", "Three"]);

    const written = await cellWrite(api, accessToken, {
      projectID: paged.id,
      cells: pagedScenes.map((scene) => ({ sceneID: scene.id, layerID: paged.layers[0].id, content: { blocks: [] } })),
    });

    await expect(cellList(api, accessToken, { projectID: paged.id, limit: 2 })).resolves.toEqual({
      cells: written.slice(0, 2),
      total: 3,
    });
    await expect(cellList(api, accessToken, { projectID: paged.id, limit: 2, offset: 2 })).resolves.toEqual({
      cells: written.slice(2),
      total: 3,
    });

    await projectDelete(api, accessToken, paged);
  });

  it("returns 400 for a page over 500 cells", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
