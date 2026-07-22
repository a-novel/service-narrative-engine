import { decodeHttpResponse, handleHttpResponse } from "@a-novel-kit/nodelib-browser/http";

import type { ZodType } from "zod";

// Fallback decoder used by fetch when no Zod validator is supplied: the body is
// trusted as-is and returned without schema validation.
async function decodeRawHttpResponse<T>(response: Response): Promise<T> {
  return await response.json();
}

/**
 * Health of a single upstream dependency reported by the service. The endpoint is
 * unauthenticated, so it carries the state alone — a failure's detail is recorded on
 * the server's traces rather than published here.
 */
export type HealthDependency = {
  status: "up" | "down";
};

/**
 * HTTP client for the narrative-engine REST API.
 *
 * It holds the service base URL and exposes the low-level request helpers that the
 * module-level `item*` functions build on. Pass an instance to those functions to
 * issue the individual endpoint calls.
 */
export class NarrativeEngineApi {
  private readonly _baseUrl: string;

  constructor(baseUrl: string) {
    this._baseUrl = baseUrl;
  }

  /**
   * Sends a request to the given path and discards the response body.
   * Throws if the server returns a non-2xx status.
   */
  async fetchVoid(input: string, init?: RequestInit): Promise<void> {
    await fetch(`${this._baseUrl}${input}`, init).then(handleHttpResponse);
  }

  /**
   * Sends a request to the given path and deserializes the JSON response body as `T`.
   * When a validator is given, the body is parsed and validated against it; otherwise
   * it is returned unchecked. Throws if the server returns a non-2xx status.
   */
  async fetch<T>(input: string, validator?: ZodType<T>, init?: RequestInit): Promise<T> {
    return await fetch(`${this._baseUrl}${input}`, init)
      .then(handleHttpResponse)
      .then(validator ? decodeHttpResponse(validator) : decodeRawHttpResponse<T>);
  }

  /** Checks that the server is reachable. Throws on any non-2xx response. */
  async ping(): Promise<void> {
    await this.fetchVoid("/ping", { method: "GET" });
  }

  /**
   * Returns the health status of every service dependency, keyed by dependency name.
   * The endpoint always responds 200; inspect each entry's `status` field to detect a
   * degraded dependency.
   */
  async health(): Promise<Record<string, HealthDependency>> {
    return await this.fetch("/healthcheck", undefined, { method: "GET" });
  }
}
