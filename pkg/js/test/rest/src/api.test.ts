import { describe, expect, it } from "vitest";

import { TemplateApi } from "@a-novel/service-template-rest";

describe("ping", () => {
  it("returns success", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expect(api.ping()).resolves.toBeUndefined();
  });
});

describe("health", () => {
  it("returns success", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expect(api.health()).resolves.toEqual({
      "client:postgres": { status: "up" },
    });
  });
});
