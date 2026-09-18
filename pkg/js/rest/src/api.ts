import { decodeHttpResponse, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import type { ZodType } from "zod";

// Parses the JSON body as-is, trusting it to match T. Used when no schema is supplied.
async function decodeRawHttpResponse<T>(response: Response): Promise<T> {
  return await response.json();
}

/** Health of one upstream dependency; failure details stay on server traces. */
export type HealthDependency = {
  status: "up" | "down";
};

/** Queue backlog returned when service-genai can measure its queue. */
export type HealthQueue = {
  pending: number;
  oldestPendingAge?: string;
};

/** Health and queue depth reported for the service-genai dependency. */
export type HealthGenAIDependency = HealthDependency & {
  queue?: HealthQueue;
};

/** Complete dependency report returned by the narrative-engine service. */
export type HealthReport = {
  "client:postgres": HealthDependency;
  "client:json-keys": HealthDependency;
  "client:genai": HealthGenAIDependency;
};

/**
 * A NarrativeEngineApi is the HTTP client for the narrative-engine service. Construct it with
 * the service base URL, then pass it to the resource helpers to make requests.
 */
export class NarrativeEngineApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  /** Sends a request and resolves once the status is validated, discarding the body. */
  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  /** Sends a request and decodes the JSON body, validating it against the schema when given. */
  async fetch<T>(input: string, validator?: ZodType<T>, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then(validator ? decodeHttpResponse(validator) : decodeRawHttpResponse<T>);
  }

  /** Liveness probe; resolves when the service is reachable. */
  async ping(): Promise<void> {
    await this.fetchVoid("/v0/ping", { method: "GET" });
  }

  /** Reports the cached health and queue metrics for every upstream dependency. */
  async health(): Promise<HealthReport> {
    return await this.fetch("/v0/healthcheck", undefined, { method: "GET" });
  }
}
