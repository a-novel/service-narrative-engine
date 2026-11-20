import { type HealthDependency, RestClient } from "@a-novel-kit/nodelib-browser/http";

export type { HealthDependency } from "@a-novel-kit/nodelib-browser/http";

/**
 * A NarrativeEngineApi is the HTTP client for the Narrative Engine service. Construct it with
 * the service base URL, then pass it to the resource helpers to make requests.
 */
export class NarrativeEngineApi extends RestClient {
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
