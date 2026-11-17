import { type ServerSentEvent, readEventStream } from "./stream";

import {
  type HealthDependency,
  RestClient,
  type RestInit,
  handleHttpResponse,
  isHttpError,
} from "@a-novel-kit/nodelib-browser/http";
import { wait } from "@a-novel-kit/nodelib-browser/utils";

export type { HealthDependency } from "@a-novel-kit/nodelib-browser/http";

/** How many attempts in a row a stream makes without receiving an event before it gives up. */
const STREAM_ATTEMPTS = 5;

/** The pause before a stream reconnects, in milliseconds. It doubles with each attempt that receives no event. */
const STREAM_RETRY_DELAY = 500;

/**
 * A NarrativeEngineApi is the HTTP client for the Narrative Engine service. Construct it with
 * the service base URL, then pass it to the resource helpers to make requests.
 */
export class NarrativeEngineApi extends RestClient {
  /**
   * Reads a server-sent event stream, and yields its events until `isLast` accepts one. A stream cut before then
   * reconnects with the last event ID it received, after a pause that doubles with each attempt that receives no
   * event, and throws the last error after 5 such attempts. It returns early once the server has nothing left to
   * stream: a 204 answer, or a response that is no event stream, such as the read a 303 answer leads to. Throws an
   * HttpError on any other non-2xx answer.
   */
  async *stream(
    input: string,
    init: RestInit,
    isLast: (event: ServerSentEvent) => boolean
  ): AsyncGenerator<ServerSentEvent> {
    let lastEventId = "";

    for (let failures = 0; ;) {
      const headers = new Headers(init.headers);
      headers.set("Accept", "text/event-stream");
      if (lastEventId) headers.set("Last-Event-ID", lastEventId);

      try {
        const response = await this.fetchResponse(input, { ...init, headers }).then(handleHttpResponse);
        if (!response.body || !response.headers.get("Content-Type")?.startsWith("text/event-stream")) {
          await response.body?.cancel();
          return;
        }

        for await (const event of readEventStream(response.body)) {
          lastEventId = event.id;
          failures = 0;
          yield event;
          if (isLast(event)) return;
        }

        throw new Error("event stream cut");
      } catch (error) {
        if (isHttpError(error) || init.signal?.aborted || ++failures >= STREAM_ATTEMPTS) throw error;
      }

      await wait(STREAM_RETRY_DELAY * 2 ** (failures - 1), init.signal);
    }
  }

  /** Liveness probe; resolves when the service is reachable. */
  async ping(): Promise<void> {
    await this.fetchVoid("/v1/ping");
  }

  /**
   * Reports the health of the service, keyed by upstream dependency name, when every dependency
   * is healthy. Throws on HTTP 503 when a dependency probe fails, or on another non-2xx response.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/v1/healthcheck");
  }
}
