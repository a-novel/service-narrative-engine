import { z } from "zod";

/**
 * Marks a block a suggestion wrote, with the model that wrote it. A block without the mark is the writer's. An editor
 * keeps the mark through the writer's edits, splits and merges. The mark is the client's declaration, not proof of
 * where the text came from: the server checks only its shape.
 */
export const GeneratedMarkSchema = z.strictObject({ model: z.string().min(1).max(200) });

/** The value {@link GeneratedMarkSchema} validates. */
export type GeneratedMark = z.infer<typeof GeneratedMarkSchema>;

/** Plain manuscript prose: action, description or exposition. */
export const NarrationBlockSchema = z.strictObject({
  kind: z.literal("narration"),
  text: z.string().max(100000),
  generated: GeneratedMarkSchema.optional(),
});

/** The value {@link NarrationBlockSchema} validates. */
export type NarrationBlock = z.infer<typeof NarrationBlockSchema>;

/** One part of a dialogue line: words the speaker says, or how they say what follows. */
export const DialoguePartSchema = z.union([
  z.strictObject({ text: z.string().max(20000) }),
  z.strictObject({ tone: z.string().max(500) }),
]);

/** The value {@link DialoguePartSchema} validates. */
export type DialoguePart = z.infer<typeof DialoguePartSchema>;

/** One spoken line of the manuscript: a speaker and ordered parts. */
export const DialogueBlockSchema = z.strictObject({
  kind: z.literal("dialogue"),
  speaker: z.string().max(200),
  parts: z.array(DialoguePartSchema).max(500),
  generated: GeneratedMarkSchema.optional(),
});

/** The value {@link DialogueBlockSchema} validates. */
export type DialogueBlock = z.infer<typeof DialogueBlockSchema>;

/** One unit of manuscript text. */
export const BlockSchema = z.discriminatedUnion("kind", [NarrationBlockSchema, DialogueBlockSchema]);

/** The value {@link BlockSchema} validates. */
export type Block = z.infer<typeof BlockSchema>;

/**
 * The content of a manuscript cell: the scene's text, as ordered blocks. It mirrors the server's
 * `content.manuscript.schema.json`, and both agree on the shared test cases.
 */
export const ManuscriptContentSchema = z.strictObject({ blocks: z.array(BlockSchema).max(10000) });

/** The value {@link ManuscriptContentSchema} validates. */
export type ManuscriptContent = z.infer<typeof ManuscriptContentSchema>;

/** The content of a notes cell: free text. It mirrors the server's `content.notes.schema.json`. */
export const NotesContentSchema = z.strictObject({ text: z.string().max(200000) });

/** The value {@link NotesContentSchema} validates. */
export type NotesContent = z.infer<typeof NotesContentSchema>;
