import { anonymousAccessToken, superAdminAccessToken, userAccessToken } from "./token";

import { afterAll, describe, expect, it, onTestFinished } from "vitest";

import { isHttpError, isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type Generation,
  GenerationKind,
  GenerationStatus,
  type GenerationSubmitRequest,
  Language,
  NarrativeEngineApi,
  type ProjectComposer,
  type Scene,
  SceneField,
  TargetMode,
  beatCreate,
  cellWrite,
  generationCancel,
  generationGet,
  generationStream,
  generationSubmit,
  projectCreate,
  projectDelete,
  sceneCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
const COLOR = "#4a90d9";

/** Reads a stream to its end. */
async function collect<T>(stream: AsyncIterable<T>): Promise<T[]> {
  const items: T[] = [];
  for await (const item of stream) items.push(item);
  return items;
}

// genai never reaches a provider in this stack, so every admitted call stays pending until cancelled.
const accessToken = await superAdminAccessToken();
const userToken = await userAccessToken();

/** Creates a project of the caller with one scene. */
async function projectWithScene(token: string, title: string): Promise<[ProjectComposer, Scene]> {
  const project = await projectCreate(api, token, { title, language: Language.Fr });
  const beat = await beatCreate(api, token, { projectID: project.id, title: "Le départ", color: COLOR });
  const scene = await sceneCreate(api, token, {
    projectID: project.id,
    beatID: beat.id,
    position: 0,
    title: "Le quai",
  });

  return [project, scene];
}

const [project, scene] = await projectWithScene(accessToken, "Les Heures claires");
afterAll(() => projectDelete(api, accessToken, project).then(() => undefined));
const manuscriptLayerID = project.layers[0].id;
const written = await cellWrite(api, accessToken, {
  projectID: project.id,
  sceneID: scene.id,
  layerID: manuscriptLayerID,
  content: {
    blocks: [
      { kind: "narration", text: "Le train entrait en gare." },
      { kind: "dialogue", speaker: "Marie", parts: [{ text: "Tu es venu." }] },
      { kind: "narration", text: "Paul ne répondit pas." },
    ],
  },
});

/** Returns a request that writes the scene's title, told apart from the others by its instruction. */
function titleRequest(instruction: string, variant?: number): GenerationSubmitRequest {
  return {
    projectID: project.id,
    kind: GenerationKind.Suggest,
    targets: [{ sceneID: scene.id, field: SceneField.Title, mode: TargetMode.Write }],
    instruction,
    variant,
  };
}

/** Submits a request as the caller, and cancels its admitted calls after the test. */
async function submit(token: string, request: GenerationSubmitRequest): Promise<Generation> {
  const generation = await generationSubmit(api, token, request);
  onTestFinished(async () => {
    for (const { generation: result } of generation.calls) {
      if (result) await generationCancel(api, token, { id: result.id });
    }
  });

  return generation;
}

/** Returns the ID of the genai generation that runs the generation's only call. */
function onlyCallID(generation: Generation): string {
  expect(generation.calls).toHaveLength(1);
  expect(generation.calls[0].generation).not.toBeNull();
  return generation.calls[0].generation!.id;
}

describe("generationSubmit", () => {
  it("submits each call with the handles that map its output back", async () => {
    const generation = await submit(accessToken, {
      projectID: project.id,
      kind: GenerationKind.Suggest,
      targets: [
        {
          sceneID: scene.id,
          layerID: manuscriptLayerID,
          mode: TargetMode.Revise,
          range: { start: 1, end: 3 },
          kept: [2],
        },
        { sceneID: scene.id, field: SceneField.Title, mode: TargetMode.Write },
      ],
      instruction: "Rends la scène plus tendue.",
    });

    expect(generation.kind).toBe("suggest");
    expect(generation.calls).toHaveLength(1);

    const [call] = generation.calls;
    expect(call.waiting).toBeUndefined();
    expect(call.generation).toEqual({ id: expect.any(String), status: GenerationStatus.Pending });
    expect(call.context.targets).toEqual([
      {
        handle: expect.stringMatching(/^M[0-9]+$/),
        sceneID: scene.id,
        layerID: manuscriptLayerID,
        mode: TargetMode.Revise,
        range: { start: 1, end: 3 },
        kept: [2],
        blocks: [
          { handle: expect.stringMatching(/^B[0-9]+$/), position: 1 },
          { handle: expect.stringMatching(/^B[0-9]+$/), position: 2 },
        ],
        beatID: null,
        override: false,
        updatedAt: expect.any(String),
      },
      {
        handle: expect.stringMatching(/^T[0-9]+$/),
        sceneID: scene.id,
        layerID: null,
        beatID: null,
        field: SceneField.Title,
        mode: TargetMode.Write,
        override: false,
        updatedAt: expect.any(String),
      },
    ]);
    // The stamps keep the server's microseconds, which a Date would drop.
    expect(new Date(call.context.targets[0].updatedAt!)).toEqual(written.updatedAt);
    expect(new Date(call.context.targets[1].updatedAt!)).toEqual(scene.updatedAt);
    expect(call.context.scenes.map(({ sceneID }) => sceneID)).toEqual([scene.id]);
    expect(call.context.layers.map(({ layerID }) => layerID)).toContain(manuscriptLayerID);
    expect(call.context.estimate).toBeGreaterThan(0);
  });

  it("replays a resent request, and runs another variant", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Replay")));

    expect(onlyCallID(await submit(accessToken, titleRequest("Replay")))).toBe(id);
    expect(onlyCallID(await submit(accessToken, titleRequest("Replay", 1)))).not.toBe(id);
  });

  it("runs a cancelled call again when resent", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Rerun")));
    await generationCancel(api, accessToken, { id });

    expect(onlyCallID(await submit(accessToken, titleRequest("Rerun")))).not.toBe(id);
  });

  it("reads overrides in place of the stored content", async () => {
    const generation = await submit(accessToken, {
      projectID: project.id,
      kind: GenerationKind.Suggest,
      targets: [{ sceneID: scene.id, layerID: manuscriptLayerID, mode: TargetMode.Revise }],
      instruction: "Override",
      overrides: {
        cells: [
          {
            sceneID: scene.id,
            layerID: manuscriptLayerID,
            content: { blocks: [{ kind: "narration", text: "Le quai était vide." }] },
          },
        ],
      },
    });

    const [target] = generation.calls[0].context.targets;
    expect(target.override).toBe(true);
    expect(target.updatedAt).toBeNull();
    expect(target.range).toEqual({ start: 0, end: 1 });
    expect(target.blocks?.map(({ position }) => position)).toEqual([0]);
  });

  it("submits a consolidation with the cell it recaps as its target", async () => {
    const generation = await submit(accessToken, {
      projectID: project.id,
      kind: GenerationKind.Consolidate,
      cell: { sceneID: scene.id, layerID: manuscriptLayerID },
    });

    expect(generation.kind).toBe(GenerationKind.Consolidate);
    expect(generation.calls[0].context.targets).toEqual([
      {
        handle: expect.stringMatching(/^T[0-9]+$/),
        sceneID: scene.id,
        layerID: manuscriptLayerID,
        beatID: null,
        override: false,
        updatedAt: expect.any(String),
      },
    ]);
  });

  it("submits a consolidation of a beat with the manuscript cells it reads", async () => {
    const generation = await submit(accessToken, {
      projectID: project.id,
      kind: GenerationKind.Consolidate,
      beatID: scene.beatID,
    });

    const [target] = generation.calls[0].context.targets;
    expect(target).toEqual({
      handle: expect.stringMatching(/^T[0-9]+$/),
      sceneID: null,
      layerID: null,
      beatID: scene.beatID,
      override: false,
      updatedAt: expect.any(String),
      sources: [{ sceneID: scene.id, cellUpdatedAt: expect.any(String) }],
    });
    expect(new Date(target.sources![0].cellUpdatedAt!)).toEqual(written.updatedAt);
  });

  it("submits a check of a range of scenes", async () => {
    const generation = await submit(accessToken, {
      projectID: project.id,
      kind: GenerationKind.Check,
      scenes: { start: 0, end: 1 },
      instruction: "Les heures.",
    });

    expect(generation.kind).toBe(GenerationKind.Check);
    expect(generation.calls[0].context.targets).toEqual([]);
    expect(generation.calls[0].context.scenes.map(({ sceneID }) => sceneID)).toEqual([scene.id]);
  });

  it("returns 400 for a consolidation with an instruction", async () => {
    const request = {
      projectID: project.id,
      kind: GenerationKind.Consolidate,
      cell: { sceneID: scene.id, layerID: manuscriptLayerID },
      instruction: "Plus court.",
    };

    await expectStatus(generationSubmit(api, accessToken, request as GenerationSubmitRequest), 400);
  });

  it("runs an admin's calls past the default tier's concurrency", async () => {
    for (const variant of [0, 1, 2]) {
      onlyCallID(await submit(accessToken, titleRequest("Uncapped", variant)));
    }
  });

  it("returns 429 naming the cap once genai admits no call", async () => {
    const [own, ownScene] = await projectWithScene(userToken, "La Ligne");
    onTestFinished(() => projectDelete(api, userToken, own).then(() => undefined));
    const request = (variant: number): GenerationSubmitRequest => ({
      projectID: own.id,
      kind: GenerationKind.Suggest,
      targets: [{ sceneID: ownScene.id, field: SceneField.Title, mode: TargetMode.Write }],
      variant,
    });

    // The default tier holds 2 calls in flight per user.
    onlyCallID(await submit(userToken, request(0)));
    onlyCallID(await submit(userToken, request(1)));

    const refusal = await generationSubmit(api, userToken, request(2)).catch((error: unknown) => error);
    expect(isHttpStatusError(refusal, 429)).toBe(true);
    expect(isHttpError(refusal) && refusal.tags.cap).toContain("concurrency");
  });

  it("returns 400 for a range past the end of the cell", async () => {
    await expectStatus(
      generationSubmit(api, accessToken, {
        projectID: project.id,
        kind: GenerationKind.Suggest,
        targets: [
          { sceneID: scene.id, layerID: manuscriptLayerID, mode: TargetMode.Revise, range: { start: 0, end: 4 } },
        ],
      }),
      400
    );
  });

  it("returns 404 for a project of another user", async () => {
    await expectStatus(generationSubmit(api, userToken, titleRequest("Elsewhere")), 404);
  });

  it("returns 401 without a session", async () => {
    await expectStatus(generationSubmit(api, "", titleRequest("Anonymous")), 401);
  });

  it("returns 403 for an anonymous session", async () => {
    await expectStatus(generationSubmit(api, await anonymousAccessToken(), titleRequest("Anonymous")), 403);
  });
});

describe("generationGet", () => {
  it("reads a call as genai holds it", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Read")));

    await expect(generationGet(api, accessToken, { id })).resolves.toEqual({ id, status: GenerationStatus.Pending });
  });

  it("returns 404 for a call of another user", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Read elsewhere")));

    await expectStatus(generationGet(api, userToken, { id }), 404);
  });

  it("returns 404 for an absent call", async () => {
    await expectStatus(generationGet(api, accessToken, { id: ABSENT_ID }), 404);
  });

  it("returns 400 for a malformed ID", async () => {
    await expectStatus(generationGet(api, accessToken, { id: "not-a-uuid" }), 400);
  });
});

describe("generationCancel", () => {
  it("cancels a call, and returns it as it stands when cancelled again", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Cancel")));
    const cancelled = { id, status: GenerationStatus.Cancelled };

    await expect(generationCancel(api, accessToken, { id })).resolves.toEqual(cancelled);
    await expect(generationCancel(api, accessToken, { id })).resolves.toEqual(cancelled);
  });

  it("returns 404 for a call of another user", async () => {
    const id = onlyCallID(await submit(accessToken, titleRequest("Cancel elsewhere")));

    await expectStatus(generationCancel(api, userToken, { id }), 404);
  });

  it("returns 404 for an absent call", async () => {
    await expectStatus(generationCancel(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("generationStream", () => {
  // genai holds a streamed submit up to 30 seconds while the provider has not begun the call.
  it("ends with the result of a call it cannot stream", { timeout: 60_000 }, async () => {
    const id = onlyCallID(await submit(accessToken, { ...titleRequest("Streamed"), stream: true }));
    await generationCancel(api, accessToken, { id });

    const events = await collect(generationStream(api, accessToken, { id }, { pollInterval: 100 }));

    expect(events).toHaveLength(1);
    expect(events[0]).toMatchObject({ type: "result", result: { id } });
    expect(events[0].type === "result" && events[0].result.status).toMatch(/^(cancelled|failed)$/);
  });

  it("returns 404 for an absent generation", async () => {
    await expectStatus(collect(generationStream(api, accessToken, { id: ABSENT_ID })), 404);
  });
});
