import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  type Layer,
  ManuscriptContentSchema,
  NarrativeEngineApi,
  type ProjectComposer,
  cellWrite,
  layerCreate,
  projectCreate,
  projectDelete,
  projectGet,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

let project: ProjectComposer;
let other: ProjectComposer;
let notes: Layer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, {
    title: "Cells",
    language: Language.En,
    scenes: [{ title: "Opening" }, { title: "Ending" }],
  });
  other = await projectCreate(api, accessToken, { title: "Other", language: Language.En, scenes: [{}] });
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
      revision: 0,
      content: {
        blocks: [
          { id: "b1", kind: "narration" as const, text: "La pluie battait les vitres." },
          { id: "b2", kind: "dialogue" as const, speaker: "Marie", parts: [{ tone: "bas" }, { text: "Tu es venu." }] },
        ],
      },
    };

    const written = await cellWrite(api, accessToken, request);
    expect(written.revision).toBe(1);
    expect(ManuscriptContentSchema.parse(written.content)).toEqual(request.content);

    const rewritten = await cellWrite(api, accessToken, { ...request, revision: 1, content: { blocks: [] } });
    expect(rewritten).toMatchObject({ revision: 2, content: { blocks: [] } });

    const stored = (await projectGet(api, accessToken, project)).cells;
    expect(stored).toContainEqual(rewritten);
  });

  it("writes a notes cell", async () => {
    const cell = await cellWrite(api, accessToken, {
      sceneID: project.scenes[1].id,
      layerID: notes.id,
      revision: 0,
      content: { text: "Marie" },
    });

    expect(cell).toMatchObject({ revision: 1, content: { text: "Marie" } });
  });

  it("returns 400 for content outside the schema of the layer's kind", async () => {
    const request = { sceneID: project.scenes[0].id, layerID: notes.id, revision: 0, content: { blocks: [] } };
    await expectStatus(cellWrite(api, accessToken, request), 400);
  });

  it("returns 404 for a scene and a layer of two projects", async () => {
    const request = { sceneID: other.scenes[0].id, layerID: notes.id, revision: 0, content: { text: "Marie" } };
    await expectStatus(cellWrite(api, accessToken, request), 404);
  });

  it("returns 409 for a stale revision", async () => {
    const request = { sceneID: project.scenes[0].id, layerID: notes.id, revision: 0, content: { text: "First" } };
    await cellWrite(api, accessToken, request);

    await expectStatus(cellWrite(api, accessToken, request), 409);
  });
});
