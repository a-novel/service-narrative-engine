import golden from "../../../../../internal/handlers/testdata/manuscript.golden.json";
import { ABSENT_ID, COLOR, accessToken, api } from "./fixtures";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  type Layer,
  LayerKind,
  ManuscriptSchema,
  type ProjectComposer,
  type Scene,
  beatsCreate,
  cellWrite,
  cellsWrite,
  layerCreate,
  manuscriptExport,
  projectCreate,
  projectDelete,
  scenesCreate,
} from "@a-novel/service-narrative-engine-rest";

// The server's handler tests answer the same golden document.
const expected = ManuscriptSchema.parse(golden);

let project: ProjectComposer;
let scenes: Scene[];
let other: ProjectComposer;
let notes: Layer;

beforeAll(async () => {
  project = await projectCreate(api, accessToken, { title: expected.title, language: Language.Fr });
  other = await projectCreate(api, accessToken, { title: "Other", language: Language.En });

  // The golden document's beats, then its scenes in one batch across the beats.
  const beats = await beatsCreate(api, accessToken, {
    projectID: project.id,
    beats: expected.appendices.beats.map((beat) => ({ ...beat, color: COLOR })),
  });
  scenes = await scenesCreate(api, accessToken, {
    projectID: project.id,
    scenes: expected.scenes.map((scene, position) => ({ beatID: beats[scene.beat].id, position, title: scene.title })),
  });

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

describe("manuscriptExport", () => {
  it("exports a French project with narration, dialogue with tones, a generated block, an empty scene, and beats", async () => {
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
      expected.scenes.map((scene, index) => ({ ...scene, blocks: index === 2 ? blocks : [] }))
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
