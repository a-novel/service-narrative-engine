import { z } from "zod";

/** Identifies a block within its cell, so edits and appendices can point at it. */
export const BlockIDSchema = z.string().regex(/^[A-Za-z0-9_-]{1,32}$/);

/** Plain manuscript prose: action, description or exposition. */
export const NarrationBlockSchema = z.strictObject({
  id: BlockIDSchema,
  kind: z.literal("narration"),
  text: z.string().max(20000),
});

/** The value {@link NarrationBlockSchema} validates. */
export type NarrationBlock = z.infer<typeof NarrationBlockSchema>;

/** One part of a dialogue line: words the speaker says, or how they say what follows. */
export const DialoguePartSchema = z.union([
  z.strictObject({ text: z.string().max(5000) }),
  z.strictObject({ tone: z.string().max(100) }),
]);

/** The value {@link DialoguePartSchema} validates. */
export type DialoguePart = z.infer<typeof DialoguePartSchema>;

/** One spoken line of the manuscript: a speaker and ordered parts. */
export const DialogueBlockSchema = z.strictObject({
  id: BlockIDSchema,
  kind: z.literal("dialogue"),
  speaker: z.string().max(100),
  parts: z.array(DialoguePartSchema).max(50),
});

/** The value {@link DialogueBlockSchema} validates. */
export type DialogueBlock = z.infer<typeof DialogueBlockSchema>;

/** One unit of manuscript text. */
export const BlockSchema = z.discriminatedUnion("kind", [NarrationBlockSchema, DialogueBlockSchema]);

/** The value {@link BlockSchema} validates. */
export type Block = z.infer<typeof BlockSchema>;

/**
 * The content of a manuscript cell: the scene's text, as ordered blocks with unique IDs. It mirrors
 * the server's `content.manuscript.schema.json`, and both agree on the shared test cases.
 */
export const ManuscriptContentSchema = z
  .strictObject({ blocks: z.array(BlockSchema).max(1000) })
  .refine((content) => new Set(content.blocks.map((block) => block.id)).size === content.blocks.length, {
    message: "block IDs are unique within a cell",
  });

/** The value {@link ManuscriptContentSchema} validates. */
export type ManuscriptContent = z.infer<typeof ManuscriptContentSchema>;

/** The content of a notes cell: free text. It mirrors the server's `content.notes.schema.json`. */
export const NotesContentSchema = z.strictObject({ text: z.string().max(20000) });

/** The value {@link NotesContentSchema} validates. */
export type NotesContent = z.infer<typeof NotesContentSchema>;
