import { beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";
import {
  NarrativeEngineApi,
  moduleListIDs,
  moduleListNamespaces,
  moduleListVersions,
  moduleSelect,
} from "@a-novel/service-narrative-engine-rest";

// Default workflows: namespaces mapped to their ordered module IDs.
// Order is determined by the file naming convention (1.idea.yaml, 2.concept.yaml, etc.).
const DEFAULT_WORKFLOWS: Record<string, string[]> = {
  agora: ["idea", "concept"],
};

let user: Awaited<ReturnType<typeof registerUser>>;

// Resolved module data: namespace -> moduleID -> { version, preversion, moduleString }.
const resolvedModules: Record<
  string,
  Record<string, { version: string; preversion?: string; moduleString: string }>
> = {};

beforeAll(async () => {
  const authApi = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const api = new NarrativeEngineApi(process.env.REST_URL!);

  const preRegister = await preRegisterUser(authApi, process.env.MAIL_HOST!);
  user = await registerUser(authApi, preRegister);

  // Resolve versions for all default modules.
  for (const [namespace, moduleIDs] of Object.entries(DEFAULT_WORKFLOWS)) {
    resolvedModules[namespace] = {};

    for (const id of moduleIDs) {
      const versions = await moduleListVersions(api, user.token.accessToken, {
        namespace,
        id,
        limit: 1,
        offset: 0,
        preversion: true,
      });

      expect(versions.length).toBe(1);

      const version = versions[0].version;
      const preversion = versions[0].preversion;

      resolvedModules[namespace][id] = {
        version,
        preversion,
        moduleString: `${namespace}:${id}@v${version}${preversion ?? ""}`,
      };
    }
  }
});

describe("moduleSelect", () => {
  it("returns all default modules", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    for (const [namespace, moduleIDs] of Object.entries(DEFAULT_WORKFLOWS)) {
      for (const id of moduleIDs) {
        const resolved = resolvedModules[namespace][id];

        const module = await moduleSelect(api, user.token.accessToken, {
          module: resolved.moduleString,
        });

        expect(module.id).toBe(id);
        expect(module.namespace).toBe(namespace);
        expect(module.version).toBe(resolved.version);
        expect(module.preversion).toBe(resolved.preversion);
        expect(module.description).toBeTruthy();
        expect(module.schema).toBeTruthy();
        expect(module.createdAt).toBeInstanceOf(Date);
      }
    }
  });

  it("returns 404 for non-existent module", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    await expectStatus(
      moduleSelect(api, user.token.accessToken, {
        module: "non-existent:module@v0.0.0",
      }),
      404
    );
  });

  it("returns 422 for invalid module string format", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    await expectStatus(
      moduleSelect(api, user.token.accessToken, {
        module: "invalid-format",
      }),
      422
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];
    const firstModuleID = DEFAULT_WORKFLOWS[firstNamespace][0];

    await expectStatus(
      moduleSelect(api, "", {
        module: resolvedModules[firstNamespace][firstModuleID].moduleString,
      }),
      401
    );
  });
});

describe("moduleListNamespaces", () => {
  it("returns all default namespaces", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const namespaces = await moduleListNamespaces(api, user.token.accessToken, {
      limit: 100,
      offset: 0,
    });

    expect(Array.isArray(namespaces)).toBe(true);

    for (const namespace of Object.keys(DEFAULT_WORKFLOWS)) {
      expect(namespaces).toContain(namespace);
    }
  });

  it("respects limit parameter", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const namespaces = await moduleListNamespaces(api, user.token.accessToken, {
      limit: 1,
      offset: 0,
    });

    expect(namespaces.length).toBeLessThanOrEqual(1);
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    await expectStatus(
      moduleListNamespaces(api, "", {
        limit: 10,
        offset: 0,
      }),
      401
    );
  });
});

describe("moduleListIDs", () => {
  it("returns all default module IDs for each namespace", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    for (const [namespace, moduleIDs] of Object.entries(DEFAULT_WORKFLOWS)) {
      const ids = await moduleListIDs(api, user.token.accessToken, {
        namespace,
        limit: 100,
        offset: 0,
      });

      expect(Array.isArray(ids)).toBe(true);

      for (const id of moduleIDs) {
        expect(ids).toContain(id);
      }
    }
  });

  it("respects limit parameter", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];

    const ids = await moduleListIDs(api, user.token.accessToken, {
      namespace: firstNamespace,
      limit: 1,
      offset: 0,
    });

    expect(ids.length).toBeLessThanOrEqual(1);
  });

  it("returns empty array for non-existent namespace", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const ids = await moduleListIDs(api, user.token.accessToken, {
      namespace: "non-existent",
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(ids)).toBe(true);
    expect(ids.length).toBe(0);
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];

    await expectStatus(
      moduleListIDs(api, "", {
        namespace: firstNamespace,
        limit: 10,
        offset: 0,
      }),
      401
    );
  });
});

describe("moduleListVersions", () => {
  it("returns versions for all default modules", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    for (const [namespace, moduleIDs] of Object.entries(DEFAULT_WORKFLOWS)) {
      for (const id of moduleIDs) {
        const versions = await moduleListVersions(api, user.token.accessToken, {
          namespace,
          id,
          limit: 10,
          offset: 0,
          preversion: true,
        });

        expect(Array.isArray(versions)).toBe(true);
        expect(versions.length).toBeGreaterThan(0);

        const version = versions[0];
        expect(version.version).toBeTruthy();
        expect(version.createdAt).toBeInstanceOf(Date);
      }
    }
  });

  it("returns empty array for non-existent module", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

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
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];
    const firstModuleID = DEFAULT_WORKFLOWS[firstNamespace][0];

    const versions = await moduleListVersions(api, user.token.accessToken, {
      namespace: firstNamespace,
      id: firstModuleID,
      limit: 1,
      offset: 0,
      preversion: true,
    });

    expect(versions.length).toBeLessThanOrEqual(1);
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];
    const firstModuleID = DEFAULT_WORKFLOWS[firstNamespace][0];

    await expectStatus(
      moduleListVersions(api, "", {
        namespace: firstNamespace,
        id: firstModuleID,
        limit: 10,
        offset: 0,
        preversion: true,
      }),
      401
    );
  });
});
