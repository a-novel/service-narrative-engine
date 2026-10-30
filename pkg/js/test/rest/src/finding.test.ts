import checkCases from "../../../../../internal/models/schemas/testdata/output.check.cases.json";

import { describe, expect, it } from "vitest";

import {
  type CheckOutput,
  CheckOutputSchema,
  FindingCategory,
  type GenerationContext,
  contentText,
  findingsVerify,
} from "@a-novel/service-narrative-engine-rest";

const SCENE_1 = "01920000-0000-7000-8000-000000000001";
const SCENE_2 = "01920000-0000-7000-8000-000000000002";

/** The context of a check call that shows two scenes. */
const context: GenerationContext = {
  targets: [],
  scenes: [
    { handle: "S1", sceneID: SCENE_1, level: "full" },
    { handle: "S2", sceneID: SCENE_2, level: "compact" },
  ],
  layers: [],
  estimate: 0,
};

/** The text the client holds for each scene. */
const texts: Record<string, string> = {
  [SCENE_1]: "Marie porte une robe bleue.\n\nElle attend  sur le quai.",
  [SCENE_2]: "Marie arrive en robe rouge.",
};

/** Returns the output of one finding, backed by the given quotes. */
function output(...evidence: CheckOutput["findings"][number]["evidence"]): CheckOutput {
  return { findings: [{ category: FindingCategory.FactsAndDetails, message: "La robe change.", evidence }] };
}

describe("CheckOutputSchema", () => {
  it("has cases to check", () => {
    expect(checkCases.length).toBeGreaterThan(0);
  });

  // The server's schema tests run the same cases, so both validators agree on each of them.
  for (const testCase of checkCases) {
    it(`agrees with the server on ${testCase.name}`, () => {
      expect(CheckOutputSchema.safeParse(testCase.document).success).toBe(testCase.valid);
    });
  }
});

describe("findingsVerify", () => {
  const sceneText = (sceneID: string): string => texts[sceneID] ?? "";

  it("keeps a finding whose quotes appear in their scenes, by scene ID", () => {
    const verified = findingsVerify(
      context,
      output({ scene: "S1", quote: "une robe bleue" }, { scene: "S2", quote: "en robe rouge" }),
      sceneText
    );

    expect(verified).toEqual([
      {
        category: FindingCategory.FactsAndDetails,
        message: "La robe change.",
        evidence: [
          { sceneID: SCENE_1, quote: "une robe bleue" },
          { sceneID: SCENE_2, quote: "en robe rouge" },
        ],
      },
    ]);
  });

  it("matches a quote across whitespace", () => {
    const quote = "bleue. Elle attend sur le quai.";

    expect(findingsVerify(context, output({ scene: "S1", quote }), sceneText)).toHaveLength(1);
  });

  it("drops a finding with a quote its scene does not hold", () => {
    const verified = findingsVerify(
      context,
      output({ scene: "S1", quote: "une robe bleue" }, { scene: "S2", quote: "une robe verte" }),
      sceneText
    );

    expect(verified).toEqual([]);
  });

  it("drops a finding that quotes the wrong scene", () => {
    expect(findingsVerify(context, output({ scene: "S2", quote: "une robe bleue" }), sceneText)).toEqual([]);
  });

  it("drops a finding that names a scene the call did not show", () => {
    expect(findingsVerify(context, output({ scene: "S9", quote: "une robe bleue" }), sceneText)).toEqual([]);
  });

  it("drops a finding with an empty quote", () => {
    expect(findingsVerify(context, output({ scene: "S1", quote: "  " }), sceneText)).toEqual([]);
  });

  it("reads each scene's text once", () => {
    const read: string[] = [];
    const counting = (sceneID: string): string => {
      read.push(sceneID);
      return sceneText(sceneID);
    };

    findingsVerify(context, output({ scene: "S1", quote: "robe" }, { scene: "S1", quote: "quai" }), counting);
    expect(read).toEqual([SCENE_1]);
  });
});

describe("contentText", () => {
  it("returns a notes cell's text", () => {
    expect(contentText({ text: "Marie, Paul" })).toBe("Marie, Paul");
  });

  it("returns a manuscript's narration and spoken words, one block per line", () => {
    const text = contentText({
      blocks: [
        { kind: "narration", text: "Le train entrait en gare." },
        { kind: "dialogue", speaker: "Marie", parts: [{ tone: "bas" }, { text: "Tu es venu" }, { text: "seul." }] },
      ],
    });

    expect(text).toBe("Le train entrait en gare.\nTu es venu seul.");
  });
});
