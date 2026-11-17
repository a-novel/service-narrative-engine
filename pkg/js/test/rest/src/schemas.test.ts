import manuscriptContentCases from "../../../../../internal/models/schemas/testdata/content.manuscript.cases.json";
import notesContentCases from "../../../../../internal/models/schemas/testdata/content.notes.cases.json";
import manuscriptCases from "../../../../../internal/models/schemas/testdata/manuscript.cases.json";
import checkCases from "../../../../../internal/models/schemas/testdata/output.check.cases.json";
import consolidateCases from "../../../../../internal/models/schemas/testdata/output.consolidate.cases.json";
import suggestCases from "../../../../../internal/models/schemas/testdata/output.suggest.cases.json";

import { describe, expect, it } from "vitest";

import {
  CheckOutputSchema,
  ManuscriptContentSchema,
  ManuscriptSchema,
  NotesContentSchema,
  RecapSchema,
  SuggestOutputSchema,
} from "@a-novel/service-narrative-engine-rest";

// The server's schema tests run the same cases, so both validators agree on each of them.
const suites = [
  ["ManuscriptContentSchema", ManuscriptContentSchema, manuscriptContentCases],
  ["NotesContentSchema", NotesContentSchema, notesContentCases],
  ["ManuscriptSchema", ManuscriptSchema, manuscriptCases],
  ["SuggestOutputSchema", SuggestOutputSchema, suggestCases],
  ["RecapSchema", RecapSchema, consolidateCases],
  ["CheckOutputSchema", CheckOutputSchema, checkCases],
] as const;

for (const [name, schema, cases] of suites) {
  describe(name, () => {
    it("has cases to check", () => {
      expect(cases.length).toBeGreaterThan(0);
    });

    for (const testCase of cases) {
      it(`agrees with the server on ${testCase.name}`, () => {
        expect(schema.safeParse(testCase.document).success).toBe(testCase.valid);
      });
    }
  });
}
