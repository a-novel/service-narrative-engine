import pluginExportsCases from "../../../../../internal/core/testdata/plugin.exports.cases.json";
import { accessToken, api, authApi } from "./fixtures";

import { createHash, randomUUID } from "node:crypto";

import { beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { anonymousAccessToken } from "@a-novel/service-authentication-rest-test";
import {
  PluginExportsSchema,
  type PluginVersionPublishRequest,
  pluginList,
  pluginManifestGet,
  pluginMessagesGet,
  pluginMigrationGet,
  pluginUIGet,
  pluginVersionList,
  pluginVersionPublish,
} from "@a-novel/service-narrative-engine-rest";

// A version the registry lacks, for what the read routes answer without one.
const absent = { plugin: "absent-plugin", version: "1.0.0" };

// The registry never removes a version, so each run publishes plugins of its own. The test stack runs without
// service tokens, so its registry takes a publication without a request token, and JSON Keys holds no API key
// to sign one with.
const run = randomUUID().slice(0, 8);
const plugin = `test-${run}`;
// A plugin that only has a prerelease, which no list returns.
const unreleased = `test-${run}-dev`;

const english = {
  title: "Test plugin",
  description: "A plugin the tests publish.",
  messages: { exports: { items: "Items" } },
};
const french = {
  title: "Greffon de test",
  description: "Un greffon que les tests publient.",
  messages: { exports: { items: "Éléments" } },
};

const migration = "export default (content) => ({ migrated: content });";

/** Returns a version of the run's plugin that the registry takes, with the members of `change` in place of its own. */
function versionOf(version: string, change: Partial<PluginVersionPublishRequest> = {}): PluginVersionPublishRequest {
  return {
    plugin,
    version,
    schema: { type: "object", default: { items: [] }, properties: { items: { type: "array" } } },
    exports: { items: { name: "exports.items", select: "$.items[*]", id: "/id", label: "/title" } },
    ui: `export default ${JSON.stringify(`ui ${version}`)};`,
    languages: { en: english },
    ...change,
  };
}

/** Publishes a version as its publisher does: as the JSON it would sign. */
async function publish(version: unknown): Promise<void> {
  await pluginVersionPublish(api, undefined, JSON.stringify(version));
}

/** Returns the checksum the registry gives a module: the SHA-256 of its bytes, in lowercase hex. */
function checksumOf(source: string): string {
  return createHash("sha256").update(source).digest("hex");
}

const first = versionOf("1.0.0");
const second = versionOf("1.1.0", { migration, languages: { en: english, fr: french } });
const prerelease = versionOf("2.0.0-dev.1");

beforeAll(async () => {
  for (const version of [first, second, prerelease, versionOf("0.1.0-dev.1", { plugin: unreleased })]) {
    await publish(version);
  }
});

describe("pluginVersionPublish", () => {
  it("publishes a version whole", async () => {
    const version = versionOf("1.0.0", { plugin: `${plugin}-whole`, migration });
    await expect(publish(version)).resolves.toBeUndefined();

    await expect(pluginManifestGet(api, accessToken, version)).resolves.toMatchObject({
      plugin: version.plugin,
      version: "1.0.0",
      schema: version.schema,
      exports: version.exports,
      hasMigration: true,
    });
  });

  it("stores neither a proof nor the ID of a key for a version published without a request token", async () => {
    const manifest = await pluginManifestGet(api, accessToken, first);
    expect(manifest.proof).toBeUndefined();
    expect(manifest.apiKeyID).toBeUndefined();
  });

  it("returns 409 for a version the registry holds, and leaves it as it is", async () => {
    await expectStatus(publish({ ...first, ui: "export default 'another module';" }), 409);

    const module = await pluginUIGet(api, accessToken, first);
    await expect(module.text()).resolves.toBe(first.ui);
  });

  it.each([
    ["a schema that uses a format", { schema: { type: "string", format: "date" } }, { schema: "pluginschema" }],
    [
      "a schema that references another document",
      { schema: { properties: { beat: { $ref: "https://example.com/beat.json" } } } },
      { schema: "pluginschema" },
    ],
    [
      "an export whose path leaves the subset",
      { exports: { items: { name: "exports.items", select: "$..items[*]", id: "/id", label: "/title" } } },
      { exports: "pluginexports" },
    ],
    ["a version without English", { languages: { fr: french } }, { languages: "pluginenglish" }],
    [
      "an English without a title",
      { languages: { en: { ...english, title: "" } } },
      { "languages[en].title": "required" },
    ],
    ["an empty UI module", { ui: "" }, { ui: "min" }],
    ["a version with build metadata", { version: "3.0.0+build.5" }, { version: "pluginversion" }],
    ["a name that is no DNS label", { plugin: "Test Plugin" }, { plugin: "dns_rfc1035_label" }],
    // No key vouches for a proof sent without a request token.
    ["a proof sent without a request token", { proof: "header.claims.signature" }, { proof: "pluginproof" }],
  ])("returns 422 naming the field for %s, and stores nothing", async (_, change, invalidFields) => {
    const refused = { ...versionOf("3.0.0"), ...change };

    await expect(publish(refused)).rejects.toMatchObject({ status: 422, tags: { invalidFields } });
    await expectStatus(pluginManifestGet(api, accessToken, { plugin, version: "3.0.0" }), 404);
  });

  it("returns 400 for a body that is no JSON", async () => {
    await expectStatus(pluginVersionPublish(api, undefined, `{"plugin": "${plugin}",`), 400);
  });

  it("returns 400 for a member the registry does not know, and stores nothing", async () => {
    // The link rules of a manifest, which the registry does not store yet.
    const withLinks = { ...versionOf("4.0.0"), links: { items: [] } };

    await expectStatus(publish(withLinks), 400);
    await expectStatus(pluginManifestGet(api, accessToken, { plugin, version: "4.0.0" }), 404);
  });
});

// The server's tests run the same cases, so both validators agree on each of them.
describe("PluginExportsSchema", () => {
  it("has cases to check", () => {
    expect(pluginExportsCases.length).toBeGreaterThan(0);
  });

  for (const testCase of pluginExportsCases) {
    it(`agrees with the server on ${testCase.name}`, () => {
      expect(PluginExportsSchema.safeParse(testCase.document).success).toBe(testCase.valid);
    });
  }
});

describe("pluginList", () => {
  it("returns a page of plugins with their total", async () => {
    const page = await pluginList(api, accessToken);
    expect(page.plugins.length).toBeGreaterThan(0);
    expect(page.total).toBeGreaterThanOrEqual(page.plugins.length);
  });

  it("lists a published plugin with its latest release, in English", async () => {
    const { plugins } = await pluginList(api, accessToken, { limit: 500 });
    expect(plugins).toContainEqual({ name: plugin, version: "1.1.0", ...english, messages: undefined });
  });

  it("reads a plugin in a language its latest release is translated in", async () => {
    const { plugins } = await pluginList(api, accessToken, { language: "fr", limit: 500 });
    expect(plugins).toContainEqual({ name: plugin, version: "1.1.0", ...french, messages: undefined });
  });

  it("leaves out a plugin that only has a prerelease", async () => {
    const { plugins } = await pluginList(api, accessToken, { limit: 500 });
    expect(plugins.map(({ name }) => name)).not.toContain(unreleased);
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
  it("returns the manifest of a published version", async () => {
    const before = Date.now();
    const manifest = await pluginManifestGet(api, accessToken, second);

    expect(manifest).toMatchObject({
      plugin,
      version: "1.1.0",
      schema: second.schema,
      exports: second.exports,
      hasMigration: true,
      uiChecksum: checksumOf(second.ui),
    });
    expect(manifest.publishedAt.getTime()).toBeLessThanOrEqual(before);
  });

  it("gives two versions with the same schema the same schema checksum", async () => {
    const [older, newer] = await Promise.all([first, second].map((v) => pluginManifestGet(api, accessToken, v)));
    expect(older.hasMigration).toBe(false);
    expect(older.schemaChecksum).toBe(newer.schemaChecksum);
  });

  it("returns a prerelease by its exact name", async () => {
    await expect(pluginManifestGet(api, accessToken, prerelease)).resolves.toMatchObject({ version: "2.0.0-dev.1" });
  });

  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginManifestGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, version: "1.0" }), 422);
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, version: "1.0.0+build" }), 422);
  });

  it("returns 422 for a name that is no DNS label", async () => {
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, plugin: "Absent Plugin" }), 422);
    await expectStatus(pluginManifestGet(api, accessToken, { ...absent, plugin: "absent-" }), 422);
  });

  it("returns 403 for an anonymous session", async () => {
    await expectStatus(pluginManifestGet(api, await anonymousAccessToken(authApi), absent), 403);
  });
});

describe("pluginUIGet", () => {
  it("returns the UI module of a published version as JavaScript", async () => {
    const module = await pluginUIGet(api, accessToken, second);
    expect(module.type).toBe("text/javascript;charset=utf-8");
    await expect(module.text()).resolves.toBe(second.ui);
  });

  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginUIGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginUIGet(api, accessToken, { ...absent, version: "latest" }), 422);
  });
});

describe("pluginMigrationGet", () => {
  it("returns the migration module of a published version", async () => {
    const module = await pluginMigrationGet(api, accessToken, second);
    await expect(module.text()).resolves.toBe(migration);
  });

  it("returns 404 for a version published without a migration", async () => {
    await expectStatus(pluginMigrationGet(api, accessToken, first), 404);
  });

  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginMigrationGet(api, accessToken, absent), 404);
  });

  it("returns 422 for a version that is no semantic version", async () => {
    await expectStatus(pluginMigrationGet(api, accessToken, { ...absent, version: "latest" }), 422);
  });
});

describe("pluginMessagesGet", () => {
  it("returns the translations of a published version in each of its languages", async () => {
    await expect(pluginMessagesGet(api, accessToken, { ...second, language: "en" })).resolves.toEqual({
      language: "en",
      ...english,
    });
    await expect(pluginMessagesGet(api, accessToken, { ...second, language: "fr" })).resolves.toEqual({
      language: "fr",
      ...french,
    });
  });

  it("returns 404 for a language the version is not translated in", async () => {
    await expectStatus(pluginMessagesGet(api, accessToken, { ...first, language: "fr" }), 404);
  });

  it("returns 404 for an absent version", async () => {
    await expectStatus(pluginMessagesGet(api, accessToken, { ...absent, language: "en" }), 404);
  });

  it("returns 422 for a language that is no BCP 47 tag", async () => {
    await expectStatus(pluginMessagesGet(api, accessToken, { ...absent, language: "french!" }), 422);
  });
});

describe("pluginVersionList", () => {
  it("returns the releases published after a version, oldest first, without the prereleases", async () => {
    const { schemaChecksum } = await pluginManifestGet(api, accessToken, second);

    await expect(pluginVersionList(api, accessToken, { plugin, after: "0.0.0" })).resolves.toEqual({
      versions: [
        { version: "1.0.0", hasMigration: false, schemaChecksum },
        { version: "1.1.0", hasMigration: true, schemaChecksum },
      ],
      total: 2,
    });
    await expect(pluginVersionList(api, accessToken, { plugin, after: "1.0.0" })).resolves.toEqual({
      versions: [{ version: "1.1.0", hasMigration: true, schemaChecksum }],
      total: 1,
    });
  });

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
