import { api } from "./fixtures";

import { describe, expect, it } from "vitest";

describe("ping", () => {
  it("returns success", async () => {
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
      "api:jsonKeys": { status: "up" },
    });
  });
});
