import { describe, expect, it } from "vitest";

import { NarrativeEngineApi } from "@a-novel/service-narrative-engine-rest";

describe("ping", () => {
  it("returns success", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
      "api:jsonKeys": { status: "up" },
    });
  });
});
