import golden from "../../../../../internal/handlers/testdata/manuscript.golden.json";
import manuscriptCases from "../../../../../internal/models/schemas/testdata/manuscript.cases.json";
import { superAdminAccessToken } from "./token";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  type Layer,
  LayerKind,
  ManuscriptSchema,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  cellWrite,
  cellsWrite,
  layerCreate,
  manuscriptExport,
  projectCreate,
  projectDelete,
  scenesCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const COLOR = "#4a90d9";

// The server's handler tests answer the same golden document.
const expected = ManuscriptSchema.parse(golden);

let project: ProjectComposer;
let scenes: Scene[];
let other: ProjectComposer;
let notes: Layer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: expected.title, language: Language.Fr });
  scenes = await scenesCreate(api, accessToken, {
    projectID: project.id,
    scenes: expected.scenes.map((scene) => ({ title: scene.title, color: COLOR })),
  });
  other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });

  // The empty scene gets no cell, and the notes cell stays out of the manuscript.
  notes = await layerCreate(api, accessToken, {
    projectID: project.id,
    position: 1,
    kind: LayerKind.Notes,
    name: "Notes",
  });
  const written = expected.scenes.flatMap((scene, index) =>
    scene.blocks.length
      ? [{ sceneID: scenes[index].id, layerID: project.layers[0].id, content: { blocks: scene.blocks } }]
      : []
  );
  await cellsWrite(api, accessToken, {
    projectID: project.id,
    cells: [...written, { sceneID: scenes[0].id, layerID: notes.id, content: { text: "Marie, Paul." } }],
  });
});

afterAll(async () => {
  for (const item of [project, other]) {
    await projectDelete(api, accessToken, item);
  }
});

describe("ManuscriptSchema", () => {
  it("has cases to check", () => {
    expect(manuscriptCases.length).toBeGreaterThan(0);
  });

  // The server's schema tests run the same cases, so both validators agree on each of them.
  for (const testCase of manuscriptCases) {
    it(`agrees with the server on ${testCase.name}`, () => {
      expect(ManuscriptSchema.safeParse(testCase.document).success).toBe(testCase.valid);
    });
  }
});

describe("manuscriptExport", () => {
  it("exports a French project with narration, dialogue with tones, and an empty scene", async () => {
    const manuscript = await manuscriptExport(api, accessToken, {
      projectID: project.id,
      layerID: project.layers[0].id,
    });
    expect(manuscript).toEqual(expected);
  });

  it("exports the manuscript layer the request names", async () => {
    const branch = await layerCreate(api, accessToken, {
      projectID: project.id,
      position: 1,
      kind: LayerKind.Manuscript,
      name: "Et si Paul restait",
    });
    const blocks = [{ kind: "narration" as const, text: "Paul resta." }];
    await cellWrite(api, accessToken, {
      projectID: project.id,
      sceneID: scenes[2].id,
      layerID: branch.id,
      content: { blocks },
    });

    const manuscript = await manuscriptExport(api, accessToken, { projectID: project.id, layerID: branch.id });
    expect(manuscript.scenes).toEqual(
      expected.scenes.map((scene, index) => ({ title: scene.title, blocks: index === 2 ? blocks : [] }))
    );
  });

  it("returns 400 for a malformed layer ID", async () => {
    await expectStatus(manuscriptExport(api, accessToken, { projectID: project.id, layerID: "not-a-uuid" }), 400);
  });

  it("returns 404 for a notes layer", async () => {
    await expectStatus(manuscriptExport(api, accessToken, { projectID: project.id, layerID: notes.id }), 404);
  });

  it("returns 404 for the manuscript layer of another project", async () => {
    await expectStatus(manuscriptExport(api, accessToken, { projectID: project.id, layerID: other.layers[0].id }), 404);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(
      manuscriptExport(api, accessToken, { projectID: ABSENT_ID, layerID: project.layers[0].id }),
      404
    );
  });
});
