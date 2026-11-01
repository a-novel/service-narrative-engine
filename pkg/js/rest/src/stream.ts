/** One server-sent event, as the event stream format of the HTML standard defines it. */
export type ServerSentEvent = {
  /** The last event ID the stream set, which a reconnect resumes after. Empty before the first one. */
  id: string;
  /** The event type, `message` when the event names none. */
  type: string;
  /** The event's data, its `data` lines joined with line feeds. */
  data: string;
};

const LF = 0x0a;
const CR = 0x0d;

/**
 * Parses an event stream body into its events, following the interpretation algorithm of the HTML standard: lines
 * end with CRLF, LF or CR, comments and unknown fields are ignored, and an event without data is dropped, as is an
 * event the body ends before.
 */
export async function* readEventStream(body: ReadableStream<Uint8Array<ArrayBuffer>>): AsyncGenerator<ServerSentEvent> {
  let id = "";
  let type = "";
  let data: string[] = [];

  // A line may span chunks: its parts wait in pending until its end arrives.
  let pending: string[] = [];
  let afterCR = false;
  let first = true;

  for await (let chunk of body.pipeThrough(new TextDecoderStream())) {
    if (first && chunk.startsWith("﻿")) chunk = chunk.slice(1);
    first = false;

    let start = 0;

    for (let index = 0; index < chunk.length; index++) {
      const code = chunk.charCodeAt(index);

      // The LF of a CRLF split across chunks ends no second line.
      if (afterCR && code === LF && index === start) {
        afterCR = false;
        start = index + 1;
        continue;
      }

      afterCR = false;
      if (code !== LF && code !== CR) continue;

      pending.push(chunk.slice(start, index));
      const line = pending.join("");
      pending = [];

      if (code === CR && index === chunk.length - 1) afterCR = true;
      else if (code === CR && chunk.charCodeAt(index + 1) === LF) index++;
      start = index + 1;

      // A blank line dispatches the event.
      if (line === "") {
        if (data.length > 0) yield { id, type: type || "message", data: data.join("\n") };
        type = "";
        data = [];
        continue;
      }

      if (line.startsWith(":")) continue;

      const colon = line.indexOf(":");
      const field = colon < 0 ? line : line.slice(0, colon);
      let value = colon < 0 ? "" : line.slice(colon + 1);
      if (value.startsWith(" ")) value = value.slice(1);

      if (field === "event") type = value;
      else if (field === "data") data.push(value);
      else if (field === "id" && !value.includes("\0")) id = value;
    }

    pending.push(chunk.slice(start));
  }
}

/** Resolves after a delay, or rejects once the signal aborts. */
export async function wait(delay: number, signal?: AbortSignal | null): Promise<void> {
  signal?.throwIfAborted();
  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(resolve, delay);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(signal.reason);
      },
      { once: true }
    );
  });
}
