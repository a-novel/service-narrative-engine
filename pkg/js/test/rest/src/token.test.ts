import { superAdminAccessToken } from "./token";

import { describe, expect, it } from "vitest";

describe("superAdminAccessToken", () => {
  it("returns a usable access token", async () => {
    const accessToken = await superAdminAccessToken();
    expect(accessToken).toBeTruthy();
  });
});
