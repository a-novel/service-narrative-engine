import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it, onTestFinished } from "vitest";

import { isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type Cell,
  type CellWriteItem,
  Language,
  type Layer,
  LayerKind,
  ManuscriptContentSchema,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  beatCreate,
  cellList,
  cellWrite,
  cellsWrite,
  layerCreate,
  projectCreate,
  projectDelete,
  scenesCreate,
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

/** Creates a project with one beat, holding one scene per title. */
async function projectWithScenes(title: string, titles: string[]): Promise<[ProjectComposer, Scene[]]> {
  const created = await projectCreate(api, accessToken, { title, language: Language.En });
  const beat = await beatCreate(api, accessToken, { projectID: created.id, title: "Beat", color: COLOR });
  const createdScenes = await scenesCreate(api, accessToken, {
    projectID: created.id,
    scenes: titles.map((sceneTitle, position) => ({ beatID: beat.id, position, title: sceneTitle })),
  });

  return [created, createdScenes];
}

/** Contents of about 200,000 characters each: a scene holds either, but not both within 400,000. */
const HALF_NOTES = { text: "a".repeat(200000) };
const HALF_MANUSCRIPT = {
  blocks: [
    { kind: "narration" as const, text: "b".repeat(100000) },
    { kind: "narration" as const, text: "c".repeat(100000) },
  ],
};

/**
 * Creates a project of one empty scene, with a notes layer after its manuscript, deleted after the test. Returns
 * the cells that write {@link HALF_NOTES} and {@link HALF_MANUSCRIPT} in that scene.
 */
async function halfCells(): Promise<{ projectID: string; cells: CellWriteItem[] }> {
  const [created, [scene]] = await projectWithScenes("Full", ["Full"]);
  onTestFinished(() => projectDelete(api, accessToken, created).then(() => undefined));

  const notesLayer = await layerCreate(api, accessToken, {
    projectID: created.id,
    position: 1,
    kind: LayerKind.Notes,
    name: "Notes",
  });

  return {
    projectID: created.id,
    cells: [
      { sceneID: scene.id, layerID: notesLayer.id, content: HALF_NOTES },
      { sceneID: scene.id, layerID: created.layers[0].id, content: HALF_MANUSCRIPT },
    ],
  };
}

/** Returns the stored cell of the test project where a scene and a layer cross. */
async function storedCell(sceneID: string, layerID: string): Promise<Cell | undefined> {
  const { cells } = await cellList(api, accessToken, { projectID: project.id, limit: 500 });

  return cells.find((cell) => cell.sceneID === sceneID && cell.layerID === layerID);
}

beforeAll(async () => {
  [project, scenes] = await projectWithScenes("Cells", ["Opening", "Ending"]);
  [other, otherScenes] = await projectWithScenes("Other", ["Only"]);
  notes = await layerCreate(api, accessToken, {
    projectID: project.id,
    position: 1,
    kind: LayerKind.Notes,
    name: "Notes",
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

    const written = await cellWrite(api, accessToken, { projectID: project.id, ...cell });
    expect(written).toEqual({ sceneID: cell.sceneID, layerID: cell.layerID, updatedAt: expect.any(Date) });
    expect(ManuscriptContentSchema.parse((await storedCell(cell.sceneID, cell.layerID))?.content)).toEqual(
      cell.content
    );

    const rewritten = await cellWrite(api, accessToken, { projectID: project.id, ...cell, content: { blocks: [] } });
    await expect(storedCell(cell.sceneID, cell.layerID)).resolves.toMatchObject({
      content: { blocks: [] },
      updatedAt: rewritten.updatedAt,
    });
  });

  it("returns 400 for content outside the schema of the layer's kind", async () => {
    const request = { projectID: project.id, sceneID: scenes[0].id, layerID: notes.id, content: { blocks: [] } };
    await expectStatus(cellWrite(api, accessToken, request), 400);
  });

  it("returns 404 for a scene of another project", async () => {
    const request = { projectID: project.id, sceneID: otherScenes[0].id, layerID: notes.id, content: { text: "M" } };
    await expectStatus(cellWrite(api, accessToken, request), 404);
  });

  it("returns 422 for a write that would bring its scene past 400,000 characters, and writes nothing", async () => {
    const { projectID, cells } = await halfCells();
    await cellWrite(api, accessToken, { projectID, ...cells[0] });
    await expectStatus(cellWrite(api, accessToken, { projectID, ...cells[1] }), 422);

    const stored = (await cellList(api, accessToken, { projectID })).cells;
    expect(stored.map((cell) => cell.layerID)).toEqual([cells[0].layerID]);
  });

  it("lets one of two concurrent writes through when together they would pass 400,000 characters", async () => {
    const { projectID, cells } = await halfCells();
    const settled = await Promise.allSettled(cells.map((cell) => cellWrite(api, accessToken, { projectID, ...cell })));

    const refused = settled.filter((result) => result.status === "rejected");
    expect(refused).toHaveLength(1);
    expect(isHttpStatusError(refused[0].reason, 422)).toBe(true);
  });
});

describe("cellsWrite", () => {
  it("writes cells of several layers at once, in the order of the request", async () => {
    const written = await cellsWrite(api, accessToken, {
      projectID: project.id,
      cells: [
        { sceneID: scenes[1].id, layerID: notes.id, content: { text: "Marie" } },
        { sceneID: scenes[1].id, layerID: project.layers[0].id, content: { blocks: [] } },
      ],
    });

    expect(written.map((cell) => cell.layerID)).toEqual([notes.id, project.layers[0].id]);
    await expect(storedCell(scenes[1].id, notes.id)).resolves.toMatchObject({ content: { text: "Marie" } });
  });

  it("returns 400 for content outside the schema of the layer's kind, and writes none of the cells", async () => {
    const cells = [
      { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Kept out" } },
      { sceneID: scenes[0].id, layerID: project.layers[0].id, content: { text: "Not a manuscript" } },
    ];
    await expectStatus(cellsWrite(api, accessToken, { projectID: project.id, cells }), 400);

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored.find((cell) => cell.sceneID === scenes[0].id && cell.layerID === notes.id)).toBeUndefined();
  });

  it("returns 400 for a scene and a layer named twice", async () => {
    const cell = { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Marie" } };
    await expectStatus(cellsWrite(api, accessToken, { projectID: project.id, cells: [cell, cell] }), 400);
  });

  it("returns 404 for a scene of another project, and writes none of the cells", async () => {
    const cells = [
      { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Kept out" } },
      { sceneID: otherScenes[0].id, layerID: notes.id, content: { text: "Marie" } },
    ];
    await expectStatus(cellsWrite(api, accessToken, { projectID: project.id, cells }), 404);

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored.find((cell) => cell.sceneID === scenes[0].id && cell.layerID === notes.id)).toBeUndefined();
  });

  it("returns 404 for a layer of another project", async () => {
    const cells = [{ sceneID: otherScenes[0].id, layerID: notes.id, content: { text: "Marie" } }];
    await expectStatus(cellsWrite(api, accessToken, { projectID: other.id, cells }), 404);
  });

  it("returns 422 for cells that would bring their scene past 400,000 characters, and writes none", async () => {
    const { projectID, cells } = await halfCells();
    await expectStatus(cellsWrite(api, accessToken, { projectID, cells }), 422);

    await expect(cellList(api, accessToken, { projectID })).resolves.toMatchObject({ cells: [] });
  });
});

describe("cellList", () => {
  it("pages through the cells of a project, with their total", async () => {
    const [paged, pagedScenes] = await projectWithScenes("Paged", ["One", "Two", "Three"]);
    onTestFinished(() => projectDelete(api, accessToken, paged).then(() => undefined));

    const written = await cellsWrite(api, accessToken, {
      projectID: paged.id,
      cells: pagedScenes.map((scene) => ({ sceneID: scene.id, layerID: paged.layers[0].id, content: { blocks: [] } })),
    });

    const keys = (cells: { sceneID: string; layerID: string }[]) => cells.map((cell) => [cell.sceneID, cell.layerID]);

    const first = await cellList(api, accessToken, { projectID: paged.id, limit: 2 });
    expect(first.total).toBe(3);
    expect(keys(first.cells)).toEqual(keys(written.slice(0, 2)));

    const second = await cellList(api, accessToken, { projectID: paged.id, limit: 2, offset: 2 });
    expect(second.total).toBe(3);
    expect(keys(second.cells)).toEqual(keys(written.slice(2)));
  });

  it("returns 400 for a page over 500 cells", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
