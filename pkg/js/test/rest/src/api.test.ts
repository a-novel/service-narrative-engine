import { describe, expect, it } from "vitest";

import { NarrativeEngineApi } from "@a-novel/service-narrative-engine-rest";

describe("ping", () => {
  it("returns success", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("reports every dependency and the generation queue", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
      "client:json-keys": { status: "up" },
      "client:genai": {
        status: "up",
        queue: { pending: 0 },
      },
    });
  });
});
