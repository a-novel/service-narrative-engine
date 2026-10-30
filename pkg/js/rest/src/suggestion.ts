import { type Block, DialogueBlockSchema, type ManuscriptContent, NarrationBlockSchema } from "./content";
import { type ContextTarget, type GenerationContext, type GenerationRange, TargetMode } from "./generation";

import { z } from "zod";

const NoteSchema = z.string().max(300);
const BlockHandleSchema = z.string().regex(/^B[0-9]+$/);
// The model writes blocks without the generated mark.
const NewBlocksSchema = z
  .array(
    z.discriminatedUnion("kind", [
      NarrationBlockSchema.omit({ generated: true }),
      DialogueBlockSchema.omit({ generated: true }),
    ])
  )
  .max(10000);

/**
 * An edit the model makes to a manuscript target, naming blocks by the handles its call showed: it replaces
 * a block, deletes one, or inserts blocks after one, or at the start of the range when its block is null.
 */
export const SuggestEditSchema = z.discriminatedUnion("op", [
  z.strictObject({ op: z.literal("replace"), block: BlockHandleSchema, blocks: NewBlocksSchema }),
  z.strictObject({ op: z.literal("insert_after"), block: BlockHandleSchema.nullable(), blocks: NewBlocksSchema }),
  z.strictObject({ op: z.literal("delete"), block: BlockHandleSchema }),
]);

/** The value {@link SuggestEditSchema} validates. */
export type SuggestEdit = z.infer<typeof SuggestEditSchema>;

/**
 * The output of a suggest call: one suggestion per target, in one list per target shape. It mirrors the
 * server's `output.suggest.schema.json`, and both agree on the shared test cases.
 */
export const SuggestOutputSchema = z.strictObject({
  manuscript: z
    .array(
      z.strictObject({
        target: z.string().regex(/^M[0-9]+$/),
        note: NoteSchema,
        edits: z.array(SuggestEditSchema).max(10000),
      })
    )
    .max(20),
  text: z
    .array(
      z.strictObject({
        target: z.string().regex(/^T[0-9]+$/),
        note: NoteSchema,
        text: z.string().max(200000),
      })
    )
    .max(20),
});

/** The value {@link SuggestOutputSchema} validates. */
export type SuggestOutput = z.infer<typeof SuggestOutputSchema>;

/**
 * An edit to a manuscript cell, at a block position of the content the call read. An insert goes before
 * the block at its position, or after the last block when its position is the cell's length.
 */
export type BlockEdit =
  | { op: "replace"; position: number; blocks: Block[] }
  | { op: "insert"; position: number; blocks: Block[] }
  | { op: "delete"; position: number };

/** The new text of a text target, a scene's title or description or a notes cell, with a note for the author. */
export type TextSuggestion = { target: ContextTarget; note: string; text: string };

/** The edits to a manuscript target, by block position, with a note for the author. */
export type ManuscriptSuggestion = { target: ContextTarget; note: string; edits: BlockEdit[] };

/** The suggestions of one call, mapped back to the targets it wrote. */
export type Suggestions = { manuscript: ManuscriptSuggestion[]; text: TextSuggestion[] };

/**
 * Maps the output of a suggest call back to the targets of its context, and the blocks it names to their
 * positions. It drops a suggestion for a target the call did not write, and an edit on a block the call did
 * not show inside the target's range, on a kept block, or on a block an earlier edit replaced or deleted. In
 * write mode, the range holds the blocks the model kept and those it wrote, so every other block of the range
 * gets a delete. A suggestion names the content it applies to by its target's `updatedAt`, or `override` for
 * content the request sent.
 */
export function suggestionsMap(context: GenerationContext, output: SuggestOutput): Suggestions {
  const targets = new Map(context.targets.map((target) => [target.handle, target]));

  return {
    manuscript: output.manuscript.flatMap(({ target: handle, note, edits }) => {
      const target = targets.get(handle);
      return target?.range ? [{ target, note, edits: blockEdits(target, target.range, edits) }] : [];
    }),
    text: output.text.flatMap(({ target: handle, note, text }) => {
      const target = targets.get(handle);
      return target ? [{ target, note, text }] : [];
    }),
  };
}

/** Maps the edits of a manuscript target to the positions of the blocks they name. */
function blockEdits(target: ContextTarget, range: GenerationRange, edits: SuggestEdit[]): BlockEdit[] {
  const positions = new Map(target.blocks?.map((block) => [block.handle, block.position]));
  const kept = new Set(target.kept);
  const taken = new Set<number>();
  const mapped: BlockEdit[] = [];

  for (const edit of edits) {
    if (edit.op === "insert_after") {
      const after = edit.block === null ? range.start - 1 : positions.get(edit.block);
      if (after !== undefined) mapped.push({ op: "insert", position: after + 1, blocks: edit.blocks });
      continue;
    }

    const position = positions.get(edit.block);
    if (position === undefined || kept.has(position) || taken.has(position)) continue;

    taken.add(position);
    mapped.push(edit.op === "replace" ? { op: "replace", position, blocks: edit.blocks } : { op: "delete", position });
  }

  if (target.mode === TargetMode.Write) {
    for (let position = range.start; position < range.end; position++) {
      if (!kept.has(position)) mapped.push({ op: "delete", position });
    }
  }

  return mapped;
}

/**
 * Applies edits to the content of a manuscript cell, and returns the new content to write. Pass some of a
 * suggestion's edits to apply only those. Positions count in the whole cell, so the edits of every call that
 * wrote a cell apply together. Inserts at one position keep their order, and an edit past the end of the
 * content, as after the cell shrank, is dropped.
 */
export function suggestionApply(content: ManuscriptContent, edits: BlockEdit[]): ManuscriptContent {
  const inserted = new Map<number, Block[]>();
  const changed = new Map<number, Block[]>();

  for (const edit of edits) {
    if (edit.op !== "insert") {
      changed.set(edit.position, edit.op === "replace" ? edit.blocks : []);
      continue;
    }

    const blocks = inserted.get(edit.position) ?? [];
    blocks.push(...edit.blocks);
    inserted.set(edit.position, blocks);
  }

  const blocks: Block[] = [];
  for (let position = 0; position <= content.blocks.length; position++) {
    blocks.push(...(inserted.get(position) ?? []));
    if (position < content.blocks.length) blocks.push(...(changed.get(position) ?? [content.blocks[position]]));
  }

  return { blocks };
}
