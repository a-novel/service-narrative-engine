import { beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";
import { NarrativeEngineApi, moduleListVersions, moduleSelect } from "@a-novel/service-narrative-engine-rest";

let user: Awaited<ReturnType<typeof registerUser>>;

const TEST_MODULE_NAMESPACE = "agora";
const TEST_MODULE_ID = "idea";

beforeAll(async () => {
  const api = new AuthenticationApi(process.env.AUTH_API_URL!);
  const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
  user = await registerUser(api, preRegister);
});

describe("moduleSelect", () => {
  let version: string;
  let preversion: string | undefined;
  let moduleString: string;

  beforeAll(async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const versions = await moduleListVersions(api, user.token.accessToken, {
      namespace: TEST_MODULE_NAMESPACE!,
      id: TEST_MODULE_ID!,
      limit: 1,
      offset: 0,
      preversion: true,
    });

    expect(versions.length).toBe(1);

    version = versions[0].version;
    preversion = versions[0].preversion;
    moduleString = `${TEST_MODULE_NAMESPACE}:${TEST_MODULE_ID}@v${version}${preversion ?? ""}`;
  });

  it("returns a module with valid module string", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const module = await moduleSelect(api, user.token.accessToken, {
      module: moduleString,
    });

    expect(module.id).toBe(TEST_MODULE_ID);
    expect(module.namespace).toBe(TEST_MODULE_NAMESPACE);
    expect(module.version).toBe(version);
    expect(module.preversion).toBe(preversion);
    expect(module.description).toBeTruthy();
    expect(module.schema).toBeTruthy();
    expect(module.ui).toBeTruthy();
    expect(module.createdAt).toBeInstanceOf(Date);
  });

  it("returns 404 for non-existent module", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      moduleSelect(api, user.token.accessToken, {
        module: "non-existent:module@v0.0.0",
      }),
      404
    );
  });

  it("returns 422 for invalid module string format", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      moduleSelect(api, user.token.accessToken, {
        module: "invalid-format",
      }),
      422
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      moduleSelect(api, "", {
        module: moduleString,
      }),
      401
    );
  });
});

describe("moduleListVersions", () => {
  it("returns a list of module versions", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const versions = await moduleListVersions(api, user.token.accessToken, {
      namespace: TEST_MODULE_NAMESPACE!,
      id: TEST_MODULE_ID!,
      limit: 10,
      offset: 0,
      preversion: true,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBeGreaterThan(0);

    const version = versions[0];
    expect(version.version).toBeTruthy();
    expect(version.createdAt).toBeInstanceOf(Date);
  });

  it("returns empty array for non-existent module", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const versions = await moduleListVersions(api, user.token.accessToken, {
      namespace: "non-existent",
      id: "module",
      limit: 10,
      offset: 0,
      preversion: true,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBe(0);
  });

  it("respects limit parameter", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const versions = await moduleListVersions(api, user.token.accessToken, {
      namespace: TEST_MODULE_NAMESPACE!,
      id: TEST_MODULE_ID!,
      limit: 1,
      offset: 0,
      preversion: true,
    });

    expect(versions.length).toBeLessThanOrEqual(1);
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      moduleListVersions(api, "", {
        namespace: TEST_MODULE_NAMESPACE!,
        id: TEST_MODULE_ID!,
        limit: 10,
        offset: 0,
        preversion: true,
      }),
      401
    );
  });
});
