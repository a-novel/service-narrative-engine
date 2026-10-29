import suggestCases from "../../../../../internal/models/schemas/testdata/output.suggest.cases.json";

import { describe, expect, it } from "vitest";

import {
  type Block,
  type ContextTarget,
  type GenerationContext,
  SceneField,
  type SuggestOutput,
  SuggestOutputSchema,
  TargetMode,
  suggestionApply,
  suggestionsMap,
} from "@a-novel/service-narrative-engine-rest";

const SCENE_ID = "01920000-0000-7000-8000-000000000001";
const LAYER_ID = "01920000-0000-7000-8000-000000000002";

/** Returns a narration block of the given text. */
function narration(text: string): Block {
  return { kind: "narration", text };
}

/** Returns a revised manuscript target over the range [1, 4) of its cell, showing blocks B1 to B3. */
function manuscriptTarget(fields: Partial<ContextTarget> = {}): ContextTarget {
  return {
    handle: "M1",
    sceneID: SCENE_ID,
    layerID: LAYER_ID,
    mode: TargetMode.Revise,
    range: { start: 1, end: 4 },
    blocks: [
      { handle: "B1", position: 1 },
      { handle: "B2", position: 2 },
      { handle: "B3", position: 3 },
    ],
    override: false,
    updatedAt: new Date("2026-10-29T00:00:00Z"),
    ...fields,
  };
}

const titleTarget: ContextTarget = {
  handle: "T2",
  sceneID: SCENE_ID,
  layerID: null,
  field: SceneField.Title,
  mode: TargetMode.Write,
  override: false,
  updatedAt: new Date("2026-10-29T00:00:00Z"),
};

/** Returns the context of a call that writes the given targets. */
function context(...targets: ContextTarget[]): GenerationContext {
  return { targets, scenes: [], layers: [], estimate: 0 };
}

/** Returns the output of one suggestion on M1, with the given edits. */
function editsOutput(edits: SuggestOutput["manuscript"][number]["edits"]): SuggestOutput {
  return { manuscript: [{ target: "M1", note: "Tightens the scene.", edits }], text: [] };
}

describe("SuggestOutputSchema", () => {
  it("has cases to check", () => {
    expect(suggestCases.length).toBeGreaterThan(0);
  });

  // The server's schema tests run the same cases, so both validators agree on each of them.
  for (const testCase of suggestCases) {
    it(`agrees with the server on ${testCase.name}`, () => {
      expect(SuggestOutputSchema.safeParse(testCase.document).success).toBe(testCase.valid);
    });
  }
});

describe("suggestionsMap", () => {
  it("maps a text suggestion to its target", () => {
    const output: SuggestOutput = {
      manuscript: [],
      text: [{ target: "T2", note: "Names the scene.", text: "Le quai" }],
    };

    expect(suggestionsMap(context(titleTarget), output)).toEqual({
      manuscript: [],
      text: [{ target: titleTarget, note: "Names the scene.", text: "Le quai" }],
    });
  });

  it("maps edits to the positions of the blocks they name", () => {
    const target = manuscriptTarget();
    const output = editsOutput([
      { op: "replace", block: "B1", blocks: [narration("first")] },
      { op: "insert_after", block: "B2", blocks: [narration("inserted")] },
      { op: "delete", block: "B3" },
    ]);

    expect(suggestionsMap(context(target), output)).toEqual({
      manuscript: [
        {
          target,
          note: "Tightens the scene.",
          edits: [
            { op: "replace", position: 1, blocks: [narration("first")] },
            { op: "insert", position: 3, blocks: [narration("inserted")] },
            { op: "delete", position: 3 },
          ],
        },
      ],
      text: [],
    });
  });

  it("inserts at the start of the range for a null block", () => {
    const output = editsOutput([{ op: "insert_after", block: null, blocks: [narration("opening")] }]);

    expect(suggestionsMap(context(manuscriptTarget()), output).manuscript[0].edits).toEqual([
      { op: "insert", position: 1, blocks: [narration("opening")] },
    ]);
  });

  it("inserts into an empty range at its position", () => {
    const target = manuscriptTarget({ range: { start: 2, end: 2 }, blocks: [] });
    const output = editsOutput([{ op: "insert_after", block: null, blocks: [narration("bridge")] }]);

    expect(suggestionsMap(context(target), output).manuscript[0].edits).toEqual([
      { op: "insert", position: 2, blocks: [narration("bridge")] },
    ]);
  });

  it("drops edits that change a kept block, and keeps inserts after it", () => {
    const target = manuscriptTarget({ kept: [2] });
    const output = editsOutput([
      { op: "replace", block: "B2", blocks: [narration("changed")] },
      { op: "delete", block: "B2" },
      { op: "insert_after", block: "B2", blocks: [narration("after")] },
    ]);

    expect(suggestionsMap(context(target), output).manuscript[0].edits).toEqual([
      { op: "insert", position: 3, blocks: [narration("after")] },
    ]);
  });

  it("drops edits on blocks the call did not show", () => {
    // B4 sits outside the range, and the call shows no block under B9.
    const output = editsOutput([
      { op: "replace", block: "B4", blocks: [narration("outside")] },
      { op: "delete", block: "B9" },
      { op: "insert_after", block: "B9", blocks: [narration("nowhere")] },
    ]);

    expect(suggestionsMap(context(manuscriptTarget()), output).manuscript[0].edits).toEqual([]);
  });

  it("drops a later edit on a block an earlier edit took", () => {
    const output = editsOutput([
      { op: "delete", block: "B1" },
      { op: "replace", block: "B1", blocks: [narration("again")] },
    ]);

    expect(suggestionsMap(context(manuscriptTarget()), output).manuscript[0].edits).toEqual([
      { op: "delete", position: 1 },
    ]);
  });

  it("drops suggestions for targets the call did not write", () => {
    const output: SuggestOutput = {
      manuscript: [{ target: "M7", note: "Elsewhere.", edits: [] }],
      text: [{ target: "T7", note: "Elsewhere.", text: "Ailleurs" }],
    };

    expect(suggestionsMap(context(manuscriptTarget(), titleTarget), output)).toEqual({ manuscript: [], text: [] });
  });

  it("deletes the other blocks of the range in write mode", () => {
    // Write mode shows the kept blocks alone.
    const target = manuscriptTarget({ mode: TargetMode.Write, kept: [2], blocks: [{ handle: "B2", position: 2 }] });
    const output = editsOutput([
      { op: "insert_after", block: null, blocks: [narration("before")] },
      { op: "insert_after", block: "B2", blocks: [narration("after")] },
    ]);

    expect(suggestionsMap(context(target), output).manuscript[0].edits).toEqual([
      { op: "insert", position: 1, blocks: [narration("before")] },
      { op: "insert", position: 3, blocks: [narration("after")] },
      { op: "delete", position: 1 },
      { op: "delete", position: 3 },
    ]);
  });
});

describe("suggestionApply", () => {
  const content = { blocks: [narration("a"), narration("b"), narration("c"), narration("d")] };

  it("applies edits at the positions of the content they were made on", () => {
    const applied = suggestionApply(content, [
      { op: "replace", position: 1, blocks: [narration("B"), narration("B'")] },
      { op: "insert", position: 3, blocks: [narration("inserted")] },
      { op: "delete", position: 3 },
      { op: "insert", position: 4, blocks: [narration("end")] },
    ]);

    expect(applied).toEqual({
      blocks: [
        narration("a"),
        narration("B"),
        narration("B'"),
        narration("c"),
        narration("inserted"),
        narration("end"),
      ],
    });
  });

  it("keeps the order of inserts at one position", () => {
    const applied = suggestionApply(content, [
      { op: "insert", position: 0, blocks: [narration("first")] },
      { op: "insert", position: 0, blocks: [narration("second")] },
    ]);

    expect(applied.blocks.slice(0, 3)).toEqual([narration("first"), narration("second"), narration("a")]);
  });

  it("drops edits past the end of the content", () => {
    const applied = suggestionApply({ blocks: [narration("a")] }, [
      { op: "replace", position: 1, blocks: [narration("gone")] },
      { op: "delete", position: 2 },
      { op: "insert", position: 2, blocks: [narration("gone")] },
      { op: "insert", position: 1, blocks: [narration("end")] },
    ]);

    expect(applied).toEqual({ blocks: [narration("a"), narration("end")] });
  });

  it("returns the content unchanged without edits", () => {
    expect(suggestionApply(content, [])).toEqual(content);
  });
});
