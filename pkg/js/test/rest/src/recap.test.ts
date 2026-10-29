import { anonymousAccessToken, superAdminAccessToken, userAccessToken } from "./token";

import { afterAll, describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  type ContextTarget,
  GenerationKind,
  type GenerationSubmitRequest,
  Language,
  NarrativeEngineApi,
  type Recap,
  beatCreate,
  beatList,
  beatRecapWrite,
  cellList,
  cellRecapWrite,
  cellWrite,
  generationCancel,
  generationSubmit,
  projectCreate,
  projectDelete,
  scenesCreate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);

const COLOR = "#4a90d9";
const RECAP: Recap = {
  condensed: "Marie attend Paul sur le quai, et le train entre en gare.",
  compact: "Marie attend Paul.",
  essentials: "Marie attend Paul sur le quai.",
  names: ["Marie", "Paul"],
};

// genai never reaches a provider in this stack, so a consolidation stays pending, and the suite writes the
// recap a client would write once it succeeded, with the stamps its context reports.
const accessToken = await superAdminAccessToken();
const userToken = await userAccessToken();

const project = await projectCreate(api, accessToken, { title: "Les Heures claires", language: Language.Fr });
afterAll(() => projectDelete(api, accessToken, project).then(() => undefined));

const beat = await beatCreate(api, accessToken, { projectID: project.id, title: "Le départ", color: COLOR });
const [scene, emptyScene] = await scenesCreate(api, accessToken, {
  projectID: project.id,
  scenes: [
    { beatID: beat.id, position: 0, title: "Le quai" },
    { beatID: beat.id, position: 1, title: "La gare" },
  ],
});
const layerID = project.layers[0].id;

/** Writes the scene's manuscript cell anew, so its updatedAt moves. */
async function writeManuscript(text: string): Promise<void> {
  await cellWrite(api, accessToken, {
    projectID: project.id,
    sceneID: scene.id,
    layerID,
    content: { blocks: [{ kind: "narration", text }] },
  });
}

await writeManuscript("Le train entrait en gare.");

/** Submits a consolidation, cancels its call after the test, and returns the target its context reports. */
async function consolidationTarget(request: GenerationSubmitRequest): Promise<ContextTarget> {
  const generation = await generationSubmit(api, accessToken, request);
  onTestFinished(async () => {
    for (const { generation: result } of generation.calls) {
      if (result) await generationCancel(api, accessToken, { id: result.id });
    }
  });

  return generation.calls[0].context.targets[0];
}

/** Returns the scene's manuscript cell as stored. */
async function storedCell() {
  const { cells } = await cellList(api, accessToken, { projectID: project.id });
  const cell = cells.find((candidate) => candidate.sceneID === scene.id && candidate.layerID === layerID);
  expect(cell).toBeDefined();
  return cell!;
}

/** Returns the recap request of the scene's manuscript cell, stamped as the target reports it. */
async function cellRecapRequest(variant?: number) {
  const target = await consolidationTarget({
    projectID: project.id,
    kind: GenerationKind.Consolidate,
    cell: { sceneID: scene.id, layerID },
    variant,
  });

  return { projectID: project.id, sceneID: scene.id, layerID, recap: RECAP, cellUpdatedAt: target.updatedAt! };
}

describe("cellRecapWrite", () => {
  it("stores the recap, fresh, without changing the cell's updatedAt", async () => {
    const before = await storedCell();
    await cellRecapWrite(api, accessToken, await cellRecapRequest());

    const after = await storedCell();
    expect(after.recap).toEqual(RECAP);
    expect(after.updatedAt).toEqual(before.updatedAt);
    expect(after.recapCellUpdatedAt).toEqual(after.updatedAt);
  });

  it("stores a recap that read the cell before it changed, stale", async () => {
    const request = await cellRecapRequest(1);
    await writeManuscript("Le train repartait.");
    await cellRecapWrite(api, accessToken, request);

    const cell = await storedCell();
    expect(cell.recap).toEqual(RECAP);
    expect(cell.recapCellUpdatedAt!.getTime()).toBeLessThan(cell.updatedAt.getTime());
  });

  it("returns 409 for a stamp older than the stored recap's", async () => {
    const request = await cellRecapRequest(2);
    await cellRecapWrite(api, accessToken, request);

    await expectStatus(cellRecapWrite(api, accessToken, { ...request, cellUpdatedAt: "2000-01-01T00:00:00Z" }), 409);
  });

  it("returns 409 for a stamp newer than the cell", async () => {
    const request = await cellRecapRequest(3);

    await expectStatus(cellRecapWrite(api, accessToken, { ...request, cellUpdatedAt: "2100-01-01T00:00:00Z" }), 409);
  });

  it("returns 400 for a recap that breaks the consolidate output schema", async () => {
    const request = await cellRecapRequest(4);

    await expectStatus(
      cellRecapWrite(api, accessToken, { ...request, recap: { condensed: "Marie." } as unknown as Recap }),
      400
    );
  });

  it("returns 404 for a project of another user, and changes nothing", async () => {
    const request = await cellRecapRequest(5);
    const before = await storedCell();

    await expectStatus(cellRecapWrite(api, userToken, { ...request, recap: { ...RECAP, essentials: "Volé." } }), 404);
    expect(await storedCell()).toEqual(before);
  });

  it("returns 404 for a cell the project lacks", async () => {
    const request = await cellRecapRequest(6);

    await expectStatus(cellRecapWrite(api, accessToken, { ...request, sceneID: emptyScene.id }), 404);
  });

  it("returns 403 for an anonymous session", async () => {
    const request = await cellRecapRequest(7);

    await expectStatus(cellRecapWrite(api, await anonymousAccessToken(), request), 403);
  });
});

describe("beatRecapWrite", () => {
  /** Returns the recap request of the beat, stamped as its consolidation's target reports it. */
  async function beatRecapRequest(variant?: number) {
    const target = await consolidationTarget({
      projectID: project.id,
      kind: GenerationKind.Consolidate,
      beatID: beat.id,
      variant,
    });

    return {
      projectID: project.id,
      beatID: beat.id,
      recap: RECAP,
      beatUpdatedAt: target.updatedAt!,
      sources: target.sources!,
    };
  }

  /** Returns the beat as stored. */
  async function storedBeat() {
    const { beats } = await beatList(api, accessToken, { projectID: project.id });
    return beats[0];
  }

  it("stores the recap with the sources the consolidation read", async () => {
    const request = await beatRecapRequest();
    expect(request.sources.map(({ sceneID }) => sceneID)).toEqual([scene.id, emptyScene.id]);
    expect(request.sources[1].cellUpdatedAt).toBeNull();

    await beatRecapWrite(api, accessToken, request);

    const stored = await storedBeat();
    expect(stored.recap).toEqual(RECAP);
    expect(stored.recapSources).toEqual(request.sources);
    expect(stored.recapBeatUpdatedAt).toEqual(stored.updatedAt);
  });

  it("returns 409 for a stamp older than the stored recap's", async () => {
    const request = await beatRecapRequest(1);
    await beatRecapWrite(api, accessToken, request);

    await expectStatus(beatRecapWrite(api, accessToken, { ...request, beatUpdatedAt: "2000-01-01T00:00:00Z" }), 409);
  });

  it("returns 404 for a project of another user, and changes nothing", async () => {
    const request = await beatRecapRequest(2);
    const before = await storedBeat();

    await expectStatus(beatRecapWrite(api, userToken, request), 404);
    expect(await storedBeat()).toEqual(before);
  });
});
