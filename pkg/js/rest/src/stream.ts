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
 * Parses an event stream body into its events, following the interpretation algorithm of the HTML standard (WHATWG
 * HTML §9.2.6, "Interpreting an event stream"): the decoder strips one leading byte order mark, lines end with CRLF,
 * LF or CR, comments and unknown fields are ignored, and an event without data is dropped, as is an event the body
 * ends before.
 *
 * Two parts of the algorithm belong to the connection, which `NarrativeEngineApi.stream` owns, and are left
 * out on purpose. The `retry` field is ignored: the server sends none, and the stream paces its reconnects itself.
 * An event without data still sets the last event ID in the standard, and here only the next event carries it: the
 * server sends no such event.
 */
export async function* readEventStream(body: ReadableStream<Uint8Array<ArrayBuffer>>): AsyncGenerator<ServerSentEvent> {
  let id = "";
  let type = "";
  // Each data line adds its value and a line feed, and the event drops the last one.
  let data = "";

  // A line may span chunks: its start waits in pending until its end arrives.
  let pending = "";
  let afterCR = false;

  for await (const chunk of body.pipeThrough(new TextDecoderStream())) {
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

      const line = pending + chunk.slice(start, index);
      pending = "";

      if (code === CR && index === chunk.length - 1) afterCR = true;
      else if (code === CR && chunk.charCodeAt(index + 1) === LF) index++;
      start = index + 1;

      // A blank line dispatches the event.
      if (line === "") {
        if (data !== "") yield { id, type: type || "message", data: data.slice(0, -1) };
        type = "";
        data = "";
        continue;
      }

      if (line.startsWith(":")) continue;

      const colon = line.indexOf(":");
      const field = colon < 0 ? line : line.slice(0, colon);
      let value = colon < 0 ? "" : line.slice(colon + 1);
      if (value.startsWith(" ")) value = value.slice(1);

      if (field === "event") type = value;
      else if (field === "data") data += value + "\n";
      else if (field === "id" && !value.includes("\0")) id = value;
    }

    pending += chunk.slice(start);
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
