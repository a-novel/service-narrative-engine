import manuscriptCases from "../../../../../internal/models/schemas/testdata/content.manuscript.cases.json";
import notesCases from "../../../../../internal/models/schemas/testdata/content.notes.cases.json";

import { describe, expect, it } from "vitest";

import { ManuscriptContentSchema, NotesContentSchema } from "@a-novel/service-narrative-engine-rest";

// The server's schema tests run the same cases, so both validators agree on each of them.
const suites = [
  ["manuscript", ManuscriptContentSchema, manuscriptCases],
  ["notes", NotesContentSchema, notesCases],
] as const;

for (const [kind, schema, cases] of suites) {
  describe(`${kind} content`, () => {
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
