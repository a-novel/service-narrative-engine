import { z } from "zod";

/**
 * Validates a string of at most `max` characters, counted as code points the way the server counts them: its
 * validator and JSON Schema's `maxLength` both do. `z.string().max` counts UTF-16 code units instead, so it would
 * refuse text the server accepts, such as a title holding emoji.
 */
export function maxChars(max: number) {
  return z.string().refine((value) => value.length <= max || codePoints(value) <= max, {
    error: `Too big: expected string to have <=${max} characters`,
  });
}

/** Counts the code points of a string. A surrogate pair, two UTF-16 code units, counts once. */
function codePoints(value: string): number {
  let count = 0;
  for (const _ of value) count++;

  return count;
}
