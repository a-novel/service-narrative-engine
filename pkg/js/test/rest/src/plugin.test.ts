import { accessToken, api, authApi } from "./fixtures";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { anonymousAccessToken } from "@a-novel/service-authentication-rest-test";
import {
  pluginList,
  pluginManifestGet,
  pluginMessagesGet,
  pluginMigrationGet,
  pluginUIGet,
  pluginVersionList,
} from "@a-novel/service-narrative-engine-rest";

// The registry takes versions from its publish route alone, which these tests cannot call yet. They cover what
// the read routes answer for a version the registry lacks, and for a request they refuse.
const absent = { plugin: "absent-plugin", version: "1.0.0" };

describe("pluginList", () => {
  it("returns a page of plugins with their total", async () => {
    const page = await pluginList(api, accessToken);
    expect(page.total).toBeGreaterThanOrEqual(page.plugins.length);
  });

  it("reads in English a language no plugin is translated in", async () => {
    await expect(pluginList(api, accessToken, { language: "de" })).resolves.toEqual(await pluginList(api, accessToken));
  });

  it("returns an empty page past the end", async () => {
    await expect(pluginList(api, accessToken, { limit: 500, offset: 1000000 })).resolves.toEqual({
      plugins: [],
      total: 0,
    });
  });

  it("returns 422 for a language that is no BCP 47 tag", async () => {
    await expectStatus(pluginList(api, accessToken, { language: "french!" }), 422);
  });

  it("returns 422 for a page over 500 plugins", async () => {
    await expectStatus(pluginList(api, accessToken, { limit: 501 }), 422);
  });

  it("returns 403 for an anonymous session", async () => {
    await expectStatus(pluginList(api, await anonymousAccessToken(authApi)), 403);
  });
});

describe("pluginManifestGet", () => {
  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginManifestGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, version: "1.0" }), 422);
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, version: "1.0.0+build" }), 422);
  });

  it("returns 403 for an anonymous session", async () => {
    await expectStatus(pluginManifestGet(api, await anonymousAccessToken(authApi), absent), 403);
  });
});

describe("pluginUIGet", () => {
  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginUIGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginUIGet(api, accessToken, { ...absent, version: "latest" }), 422);
  });
});

describe("pluginMigrationGet", () => {
  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginMigrationGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginMigrationGet(api, accessToken, { ...absent, version: "latest" }), 422);
  });
});

describe("pluginMessagesGet", () => {
  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginMessagesGet(api, accessToken, { ...absent, language: "en" }), 404);
  });

  it("returns 422 for a language that is no BCP 47 tag", async () => {
    await expectStatus(pluginMessagesGet(api, accessToken, { ...absent, language: "french!" }), 422);
  });
});

describe("pluginVersionList", () => {
  it("returns an empty page for a plugin the registry lacks", async () => {
    await expect(pluginVersionList(api, accessToken, { plugin: absent.plugin, after: "1.0.0" })).resolves.toEqual({
      versions: [],
      total: 0,
    });
  });

  it("takes a prerelease to start after", async () => {
    await expect(
      pluginVersionList(api, accessToken, { plugin: absent.plugin, after: "1.0.0-dev.1", limit: 500, offset: 5 })
    ).resolves.toEqual({ versions: [], total: 0 });
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginVersionList(api, accessToken, { plugin: absent.plugin, after: "1.0" }), 422);
  });

  it("returns 422 for a page over 500 releases", async () => {
    await expectStatus(pluginVersionList(api, accessToken, { plugin: absent.plugin, after: "1.0.0", limit: 501 }), 422);
  });
});
