import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type Cell,
  Language,
  type Layer,
  ManuscriptContentSchema,
  NarrativeEngineApi,
  type ProjectComposer,
  cellList,
  cellWrite,
  layerCreate,
  projectCreate,
  projectDelete,
  projectGet,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const accessToken = await superAdminAccessToken();

let project: ProjectComposer;
let other: ProjectComposer;
let notes: Layer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, {
    title: "Cells",
    language: Language.En,
    scenes: [
      { title: "Opening", color: "#4a90d9" },
      { title: "Ending", color: "#4a90d9" },
    ],
  });
  other = await projectCreate(api, accessToken, {
    title: "Other",
    language: Language.En,
    scenes: [{ title: "Only", color: "#4a90d9" }],
  });
  notes = await layerCreate(api, accessToken, { projectID: project.id, after: project.layers[0].id, name: "Notes" });
});

afterAll(async () => {
  for (const item of [project, other]) {
    await projectDelete(api, accessToken, await projectGet(api, accessToken, item));
  }
});

describe("cellWrite", () => {
  it("writes a manuscript cell, then rewrites it", async () => {
    const request = {
      sceneID: project.scenes[0].id,
      layerID: project.layers[0].id,
      content: {
        blocks: [
          { id: "b1", kind: "narration" as const, text: "La pluie battait les vitres." },
          { id: "b2", kind: "dialogue" as const, speaker: "Marie", parts: [{ tone: "bas" }, { text: "Tu es venu." }] },
        ],
      },
    };

    const written = await cellWrite(api, accessToken, request);
    expect(ManuscriptContentSchema.parse(written.content)).toEqual(request.content);

    const rewritten = await cellWrite(api, accessToken, { ...request, content: { blocks: [] } });
    expect(rewritten.content).toEqual({ blocks: [] });

    const stored = (await cellList(api, accessToken, { projectID: project.id })).cells;
    expect(stored).toContainEqual(rewritten);
  });

  it("writes a notes cell", async () => {
    const cell = await cellWrite(api, accessToken, {
      sceneID: project.scenes[1].id,
      layerID: notes.id,
      content: { text: "Marie" },
    });

    expect(cell.content).toEqual({ text: "Marie" });
  });

  it("returns 400 for content outside the schema of the layer's kind", async () => {
    const request = { sceneID: project.scenes[0].id, layerID: notes.id, content: { blocks: [] } };
    await expectStatus(cellWrite(api, accessToken, request), 400);
  });

  it("returns 404 for a scene and a layer of two projects", async () => {
    const request = { sceneID: other.scenes[0].id, layerID: notes.id, content: { text: "Marie" } };
    await expectStatus(cellWrite(api, accessToken, request), 404);
  });
});

describe("cellList", () => {
  it("pages through the cells of a project, with their total", async () => {
    const paged = await projectCreate(api, accessToken, {
      title: "Paged",
      language: Language.En,
      scenes: [
        { title: "One", color: "#4a90d9" },
        { title: "Two", color: "#4a90d9" },
        { title: "Three", color: "#4a90d9" },
      ],
    });

    const written: Cell[] = [];
    for (const scene of paged.scenes) {
      const request = { sceneID: scene.id, layerID: paged.layers[0].id, content: { blocks: [] } };
      written.push(await cellWrite(api, accessToken, request));
    }

    const first = await cellList(api, accessToken, { projectID: paged.id, limit: 2 });
    expect(first.total).toBe(3);
    expect(first.cells).toHaveLength(2);

    const last = first.cells[1];
    const second = await cellList(api, accessToken, {
      projectID: paged.id,
      limit: 2,
      afterSceneID: last.sceneID,
      afterLayerID: last.layerID,
    });
    expect(second.total).toBe(3);

    const all = [...first.cells, ...second.cells];
    expect(all).toHaveLength(3);
    expect(all).toEqual(expect.arrayContaining(written));

    await projectDelete(api, accessToken, paged);
  });

  it("returns 400 for a page over 500 cells", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: project.id, limit: 501 }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(cellList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});
