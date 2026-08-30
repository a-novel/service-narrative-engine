import { otherSuperAdminAccessToken, superAdminAccessToken } from "./token";

import { describe, expect, it } from "vitest";

import { HTTP_HEADERS, isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import {
  type Generation,
  NarrativeEngineApi,
  generationGet,
  generationSubmit,
  ideaHistory,
  ideaVersionCreate,
  manuscriptCreate,
  manuscriptHistory,
  projectCreate,
  projectGet,
  stepValueCreate,
  stepValueHistory,
} from "@a-novel/service-narrative-engine-rest";

const terminalStatuses = new Set(["succeeded", "failed", "abandoned", "cancelled"]);

async function expectHttpStatus(promise: Promise<unknown>, status: number): Promise<void> {
  try {
    await promise;
  } catch (error) {
    expect(isHttpStatusError(error, status)).toBe(true);

    return;
  }

  throw new Error(`expected HTTP ${status}`);
}

async function waitForGeneration(
  api: NarrativeEngineApi,
  accessToken: string,
  projectID: string,
  initial: Generation
): Promise<Generation> {
  let generation = initial;
  const deadline = Date.now() + 20_000;

  while (!terminalStatuses.has(generation.status)) {
    if (Date.now() >= deadline) throw new Error("generation did not settle before the deadline");

    await new Promise((resolve) => setTimeout(resolve, 100));
    generation = await generationGet(api, accessToken, projectID, generation.id);
  }

  return generation;
}

describe("owned Project contract", () => {
  it("round-trips bounded content and forwards only client-composed Generation input", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const [ownerToken, otherOwnerToken] = await Promise.all([superAdminAccessToken(), otherSuperAdminAccessToken()]);

    const project = await projectCreate(api, ownerToken, {
      idea: {
        title: "The Answering Light",
        genre: "speculative",
        seed: "A lighthouse keeper hears a foghorn answer from beneath the sea.",
      },
    });

    await expectHttpStatus(
      api.fetch(`/v0/projects/${project.id}/ideas`, undefined, {
        method: "POST",
        headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${ownerToken}` },
        body: JSON.stringify({
          value: { title: "", genre: "speculative", seed: "Still nonblank." },
        }),
      }),
      422
    );

    await expectHttpStatus(
      api.fetch(`/v0/projects/${project.id}/manuscripts`, undefined, {
        method: "POST",
        headers: { ...HTTP_HEADERS.JSON, Authorization: `Bearer ${ownerToken}` },
        body: JSON.stringify({
          value: { blocks: [{ type: "media", metadata: {}, data: {} }] },
        }),
      }),
      422
    );

    const formerEngineViolation = ["an", "array", "where", "an", "object", "was", "expected"];
    await stepValueCreate(api, ownerToken, {
      projectID: project.id,
      key: "formerly-invalid",
      value: formerEngineViolation,
    });

    for (let revision = 0; revision < 26; revision += 1) {
      await Promise.all([
        ideaVersionCreate(api, ownerToken, {
          projectID: project.id,
          value: {
            title: `The Answering Light ${revision}`,
            genre: "speculative",
            seed: `The submerged signal moves closer in revision ${revision}.`,
          },
        }),
        stepValueCreate(api, ownerToken, {
          projectID: project.id,
          key: "outline",
          value: { revision, arbitraryClientShape: { beat: `beat-${revision}` } },
        }),
        manuscriptCreate(api, ownerToken, {
          projectID: project.id,
          value: {
            blocks: [
              {
                type: "text",
                metadata: { revision, editor: { folded: false } },
                data: { text: `Paragraph revision ${revision}.`, marks: [] },
              },
            ],
          },
        }),
      ]);
    }

    const [snapshot, ideas, steps, manuscripts, formerlyInvalid] = await Promise.all([
      projectGet(api, ownerToken, project.id),
      ideaHistory(api, ownerToken, project.id),
      stepValueHistory(api, ownerToken, project.id, "outline"),
      manuscriptHistory(api, ownerToken, project.id),
      stepValueHistory(api, ownerToken, project.id, "formerly-invalid"),
    ]);

    expect(ideas).toHaveLength(25);
    expect(steps).toHaveLength(25);
    expect(manuscripts).toHaveLength(25);
    expect(ideas[0]?.value.title).toBe("The Answering Light 25");
    expect(steps[0]?.value).toEqual({
      revision: 25,
      arbitraryClientShape: { beat: "beat-25" },
    });
    expect(manuscripts[0]?.value.blocks[0]?.data.text).toBe("Paragraph revision 25.");
    expect(formerlyInvalid[0]?.value).toEqual(formerEngineViolation);
    expect(snapshot.idea.id).toBe(ideas[0]?.id);
    expect(snapshot.manuscript?.id).toBe(manuscripts[0]?.id);
    expect(snapshot.stepValues.find(({ key }) => key === "outline")?.id).toBe(steps[0]?.id);
    expect(snapshot.stepValues.find(({ key }) => key === "formerly-invalid")?.value).toEqual(formerEngineViolation);

    const instructions = "CLIENT_INSTRUCTIONS_MARKER: continue without resolving the mystery.";
    const input = { partialParagraph: "CLIENT_INPUT_MARKER: The lens went dark." };
    const context = {
      clientContextMarker: "CLIENT_CONTEXT_MARKER",
      deliberatelySelected: { outlineRevision: 7 },
    };
    const outputSchema = {
      type: "object",
      additionalProperties: false,
      required: ["accepted", "source", "paragraph"],
      properties: {
        accepted: { type: "boolean" },
        source: { type: "string" },
        paragraph: { type: "string" },
      },
    };

    const submitted = await generationSubmit(api, ownerToken, {
      projectID: project.id,
      idempotencyKey: "integration-generation-1",
      instructions,
      input,
      context,
      outputSchema,
    });
    const generation = await waitForGeneration(api, ownerToken, project.id, submitted);

    expect(generation.status).toBe("succeeded");
    expect(generation.proposal).toEqual({
      accepted: true,
      source: "controlled-provider",
      paragraph: "The answering light moved beneath the waves.",
    });

    const replay = await generationSubmit(api, ownerToken, {
      projectID: project.id,
      idempotencyKey: "integration-generation-1",
      instructions,
      input,
      context,
      outputSchema,
    });
    expect(replay.id).toBe(generation.id);

    await expectHttpStatus(
      generationSubmit(api, ownerToken, {
        projectID: project.id,
        idempotencyKey: "integration-generation-1",
        instructions: `${instructions} changed`,
        input,
        context,
        outputSchema,
      }),
      409
    );

    const providerResponse = await fetch(`${process.env.OPENAI_MOCK_URL!}/requests/latest`);
    expect(providerResponse.ok).toBe(true);

    const providerRequest = (await providerResponse.json()) as {
      model: string;
      reasoning: { effort: string };
      max_output_tokens: number;
      safety_identifier: string;
      instructions: string;
      input: string;
      text: { format: { schema: unknown } };
      background: boolean;
      store: boolean;
    } & Record<string, unknown>;
    const providerInput = JSON.parse(providerRequest.input) as Record<string, unknown>;

    expect(providerRequest.model).toBe("gpt-5.6-terra");
    expect(providerRequest.reasoning.effort).toBe("medium");
    expect(providerRequest.max_output_tokens).toBe(32_768);
    expect(providerRequest.safety_identifier).toMatch(/^[0-9a-f]{64}$/u);
    expect(providerRequest.instructions).toBe(instructions);
    expect(providerInput).toEqual({ input, context });
    expect(Object.keys(providerInput).sort()).toEqual(["context", "input"]);
    expect(providerRequest.text.format.schema).toEqual(outputSchema);
    expect(providerRequest.background).toBe(true);
    expect(providerRequest.store).toBe(false);
    expect(providerRequest).not.toHaveProperty("tools");

    const selected = await stepValueCreate(api, ownerToken, {
      projectID: project.id,
      key: "selected-proposal",
      value: generation.proposal,
    });
    expect(selected.value).toEqual(generation.proposal);

    await Promise.all([
      expectHttpStatus(projectGet(api, otherOwnerToken, project.id), 404),
      expectHttpStatus(ideaHistory(api, otherOwnerToken, project.id), 404),
      expectHttpStatus(
        ideaVersionCreate(api, otherOwnerToken, {
          projectID: project.id,
          value: project.idea.value,
        }),
        404
      ),
      expectHttpStatus(stepValueHistory(api, otherOwnerToken, project.id, "outline"), 404),
      expectHttpStatus(
        stepValueCreate(api, otherOwnerToken, {
          projectID: project.id,
          key: "forbidden",
          value: { shouldNotPersist: true },
        }),
        404
      ),
      expectHttpStatus(manuscriptHistory(api, otherOwnerToken, project.id), 404),
      expectHttpStatus(
        manuscriptCreate(api, otherOwnerToken, {
          projectID: project.id,
          value: manuscripts[0]!.value,
        }),
        404
      ),
      expectHttpStatus(
        generationSubmit(api, otherOwnerToken, {
          projectID: project.id,
          idempotencyKey: "other-owner-forbidden",
          instructions,
          input,
          context,
          outputSchema,
        }),
        404
      ),
      expectHttpStatus(generationGet(api, otherOwnerToken, project.id, generation.id), 404),
    ]);
  }, 60_000);
});
