import type { ManuscriptContent, NotesContent } from "./content";
import type { GenerationContext } from "./generation";

import { z } from "zod";

/** The kinds of coherence issue a check reports. */
export enum FindingCategory {
  /** A contradiction in the timeline or the plot. */
  TimelineAndPlot = "timeline-and-plot",
  /** A contradiction in how a character acts or speaks. */
  Characterization = "characterization",
  /** A contradiction in the rules of the story's world. */
  WorldBuilding = "world-building",
  /** A contradiction in a fact or a detail, such as a name or a number. */
  FactsAndDetails = "facts-and-details",
  /** A contradiction in the style, such as the voice or the tense. */
  Style = "style",
}

/** Validates a {@link FindingCategory}. */
export const FindingCategorySchema = z.enum([
  FindingCategory.TimelineAndPlot,
  FindingCategory.Characterization,
  FindingCategory.WorldBuilding,
  FindingCategory.FactsAndDetails,
  FindingCategory.Style,
]);

/**
 * The output of a check call: the possible coherence issues it finds, each backed by quotes of the scenes it
 * showed, named by their handles. It mirrors the server's `output.check.schema.json`, and both agree on the
 * shared test cases.
 */
export const CheckOutputSchema = z.strictObject({
  findings: z
    .array(
      z.strictObject({
        category: FindingCategorySchema,
        message: z.string().max(1000),
        evidence: z
          .array(z.strictObject({ scene: z.string().regex(/^S[0-9]+$/), quote: z.string().max(1000) }))
          .min(1)
          .max(10),
      })
    )
    .max(50),
});

/** The value {@link CheckOutputSchema} validates. */
export type CheckOutput = z.infer<typeof CheckOutputSchema>;

/** A passage of a scene that shows a finding. */
export type FindingEvidence = { sceneID: string; quote: string };

/** A coherence issue a check found, with its quotes mapped to their scenes. */
export type Finding = { category: FindingCategory; message: string; evidence: FindingEvidence[] };

/**
 * Maps the findings of a check call to the scenes of its context, and keeps only those whose every quote
 * appears word for word in the text the client holds for its scene, after whitespace normalization. A quote
 * that names a scene the call did not show, or that its scene does not hold, drops its finding. `sceneText`
 * returns every text the client holds for a scene, such as its cells through {@link contentText}, and the
 * versions of their recaps.
 */
export function findingsVerify(
  context: GenerationContext,
  output: CheckOutput,
  sceneText: (sceneID: string) => string
): Finding[] {
  const scenes = new Map(context.scenes.map(({ handle, sceneID }) => [handle, sceneID]));
  const texts = new Map<string, string>();

  const holds = (sceneID: string, quote: string): boolean => {
    let text = texts.get(sceneID);
    if (text === undefined) {
      text = normalizeSpace(sceneText(sceneID));
      texts.set(sceneID, text);
    }

    const normalized = normalizeSpace(quote);
    return normalized !== "" && text.includes(normalized);
  };

  return output.findings.flatMap(({ category, message, evidence }) => {
    const verified = evidence.flatMap(({ scene, quote }) => {
      const sceneID = scenes.get(scene);
      return sceneID !== undefined && holds(sceneID, quote) ? [{ sceneID, quote }] : [];
    });

    return verified.length === evidence.length ? [{ category, message, evidence: verified }] : [];
  });
}

/**
 * Returns the text of a cell's content a quote can come from: a notes cell's text, or a manuscript's blocks,
 * one per line, each dialogue as its spoken words.
 */
export function contentText(content: ManuscriptContent | NotesContent): string {
  if ("text" in content) return content.text;

  return content.blocks
    .map((block) =>
      block.kind === "narration"
        ? block.text
        : block.parts.flatMap((part) => ("text" in part ? [part.text] : [])).join(" ")
    )
    .join("\n");
}

/** Collapses every run of whitespace to one space, and trims the ends. */
function normalizeSpace(text: string): string {
  return text.replace(/\s+/g, " ").trim();
}
