import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";
import {
  NarrativeEngineApi,
  moduleListVersions,
  projectDelete,
  projectInit,
  schemaCreate,
  schemaGenerate,
  schemaListVersions,
  schemaRewrite,
  schemaSelect,
} from "@a-novel/service-narrative-engine-rest";

// Default workflows: namespaces mapped to their ordered module IDs.
// Order is determined by the file naming convention (1.idea.yaml, 2.concept.yaml, etc.).
const DEFAULT_WORKFLOWS: Record<string, string[]> = {
  agora: ["idea", "concept"],
};

let user: Awaited<ReturnType<typeof registerUser>>;
let project: Awaited<ReturnType<typeof projectInit>>;

let api: NarrativeEngineApi;

// First module used for non-AI tests (schema CRUD, etc.).
let moduleString: string;

// Resolved module strings for all default modules, preserving order.
// Maps namespace -> moduleID -> resolved module string.
const resolvedModules: Record<string, Record<string, string>> = {};

// Flat-ordered list of all resolved module strings (for workflow creation).
const allModuleStrings: string[] = [];

beforeAll(async () => {
  const authApi = new AuthenticationApi(process.env.AUTH_API_URL!);
  api = new NarrativeEngineApi(process.env.API_URL!);

  const preRegister = await preRegisterUser(authApi, process.env.MAIL_TEST_HOST!);
  user = await registerUser(authApi, preRegister);

  // Resolve module strings for all default modules.
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
      const resolved = `${namespace}:${id}@v${version}${preversion ?? ""}`;

      resolvedModules[namespace][id] = resolved;
      allModuleStrings.push(resolved);
    }
  }

  // Use the first module for non-AI tests.
  moduleString = allModuleStrings[0];
});

beforeEach(async () => {
  project = await projectInit(api, user.token.accessToken, {
    lang: "en",
    title: `Schema Test Project ${Date.now()}`,
    workflow: [moduleString],
  });
});

afterEach(async () => {
  await projectDelete(api, user.token.accessToken, { id: project.id });
});

describe("schemaCreate", () => {
  it("creates a new schema", async () => {
    const schemaId = crypto.randomUUID();
    const schema = await schemaCreate(api, user.token.accessToken, {
      id: schemaId,
      projectID: project.id,
      module: moduleString,
      source: "USER",
      data: { test: "data" },
    });

    expect(schema.id).toBe(schemaId);
    expect(schema.projectID).toBe(project.id);
    expect(schema.module).toBe(moduleString);
    expect(schema.source).toBe("USER");
    expect(schema.data).toEqual({ test: "data" });
    expect(schema.createdAt).toBeInstanceOf(Date);
  });

  it("creates schema with AI source", async () => {
    const schema = await schemaCreate(api, user.token.accessToken, {
      id: crypto.randomUUID(),
      projectID: project.id,
      module: moduleString,
      source: "AI",
      data: { generated: true },
    });

    expect(schema.source).toBe("AI");
  });

  it("returns 404 for non-existent project", async () => {
    await expectStatus(
      schemaCreate(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        projectID: crypto.randomUUID(),
        module: moduleString,
        source: "USER",
        data: {},
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    await expectStatus(
      schemaCreate(api, "", {
        id: crypto.randomUUID(),
        projectID: crypto.randomUUID(),
        module: moduleString,
        source: "USER",
        data: {},
      }),
      401
    );
  });
});

describe("schemaSelect", () => {
  it("selects a schema by id", async () => {
    const schemaId = crypto.randomUUID();
    await schemaCreate(api, user.token.accessToken, {
      id: schemaId,
      projectID: project.id,
      module: moduleString,
      source: "USER",
      data: { select: "test" },
    });

    const schema = await schemaSelect(api, user.token.accessToken, {
      id: schemaId,
      projectID: project.id,
    });

    expect(schema.id).toBe(schemaId);
    expect(schema.data).toEqual({ select: "test" });
  });

  it("selects a schema by module", async () => {
    await schemaCreate(api, user.token.accessToken, {
      id: crypto.randomUUID(),
      projectID: project.id,
      module: moduleString,
      source: "USER",
      data: { module: "select" },
    });

    const schema = await schemaSelect(api, user.token.accessToken, {
      projectID: project.id,
      module: moduleString,
    });

    // Schema select returns the latest schema for the module, which may have a different version.
    // Check that the module namespace and ID match.
    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];
    const firstModuleID = DEFAULT_WORKFLOWS[firstNamespace][0];
    expect(schema.module).toContain(`${firstNamespace}:${firstModuleID}@v`);
  });

  it("returns 404 for non-existent schema", async () => {
    await expectStatus(
      schemaSelect(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        projectID: project.id,
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      schemaSelect(api, "", {
        projectID: crypto.randomUUID(),
      }),
      401
    );
  });
});

describe("schemaRewrite", () => {
  it("rewrites schema data", async () => {
    const schemaId = crypto.randomUUID();
    await schemaCreate(api, user.token.accessToken, {
      id: schemaId,
      projectID: project.id,
      module: moduleString,
      source: "USER",
      data: { original: "data" },
    });

    const updatedSchema = await schemaRewrite(api, user.token.accessToken, {
      id: schemaId,
      data: { updated: "data" },
    });

    expect(updatedSchema.id).toBe(schemaId);
    expect(updatedSchema.data).toEqual({ updated: "data" });
  });

  it("returns 404 for non-existent schema", async () => {
    await expectStatus(
      schemaRewrite(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        data: { test: "data" },
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    await expectStatus(
      schemaRewrite(api, "", {
        id: crypto.randomUUID(),
        data: {},
      }),
      401
    );
  });
});

describe("schemaListVersions", () => {
  it("returns a list of schema versions", async () => {
    // Create initial schema
    const schemaId = crypto.randomUUID();
    await schemaCreate(api, user.token.accessToken, {
      id: schemaId,
      projectID: project.id,
      module: moduleString,
      source: "USER",
      data: { version: 1 },
    });

    // Rewrite to create a new version
    await schemaRewrite(api, user.token.accessToken, {
      id: schemaId,
      data: { version: 2 },
    });

    const firstNamespace = Object.keys(DEFAULT_WORKFLOWS)[0];
    const firstModuleID = DEFAULT_WORKFLOWS[firstNamespace][0];

    const versions = await schemaListVersions(api, user.token.accessToken, {
      projectID: project.id,
      moduleID: firstModuleID,
      moduleNamespace: firstNamespace,
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBeGreaterThan(0);

    const version = versions[0];
    expect(version.id).toBeTruthy();
    expect(version.createdAt).toBeInstanceOf(Date);
  });

  it("returns empty array for non-existent module", async () => {
    const versions = await schemaListVersions(api, user.token.accessToken, {
      projectID: project.id,
      moduleID: "non-existent",
      moduleNamespace: "non-existent",
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBe(0);
  });

  it("returns 401 without access token", async () => {
    await expectStatus(
      schemaListVersions(api, "", {
        projectID: crypto.randomUUID(),
        moduleID: "test",
        moduleNamespace: "test",
        limit: 10,
        offset: 0,
      }),
      401
    );
  });
});

describe(
  "schemaGenerate",
  () => {
    // Generates all default modules in order for each namespace, verifying the full workflow.
    describe(
      "modules generation",
      async () => {
        for (const [namespace, moduleIDs] of Object.entries(DEFAULT_WORKFLOWS)) {
          describe(
            `for namespace ${namespace}`,
            async () => {
              let nsProject: Awaited<ReturnType<typeof projectInit>>;

              beforeAll(async () => {
                nsProject = await projectInit(api, user.token.accessToken, {
                  lang: "en",
                  title: `Schema Workflow Test ${Date.now()}`,
                  workflow: Object.values(resolvedModules[namespace]),
                });
              });

              afterAll(async () => {
                await projectDelete(api, user.token.accessToken, { id: nsProject.id });
              });

              for (const id of moduleIDs) {
                it(`generates module ${id}`, async () => {
                  const modString = resolvedModules[namespace][id];

                  const schema = await schemaGenerate(api, user.token.accessToken, {
                    projectID: nsProject.id,
                    module: modString,
                    lang: "en",
                  });

                  expect(schema.id).toBeTruthy();
                  expect(schema.projectID).toBe(nsProject.id);
                  expect(schema.module).toContain(`${namespace}:${id}@v`);
                  expect(schema.source).toBe("AI");
                  expect(schema.data).toBeTruthy();
                  expect(schema.createdAt).toBeInstanceOf(Date);
                }, 60000);
              }
            },
            60000 * moduleIDs.length
          );
        }
      },
      60000 * Object.keys(DEFAULT_WORKFLOWS).length
    );

    it("generates a schema with instruction", async () => {
      const schema = await schemaGenerate(api, user.token.accessToken, {
        projectID: project.id,
        module: moduleString,
        lang: "en",
        instruction: "Make it about music.",
      });

      expect(schema.id).toBeTruthy();
      expect(schema.projectID).toBe(project.id);
      expect(schema.source).toBe("AI");
      expect(schema.data).toBeTruthy();
      expect(schema.createdAt).toBeInstanceOf(Date);
    }, 60000);

    it("returns 404 for non-existent project", async () => {
      await expectStatus(
        schemaGenerate(api, user.token.accessToken, {
          projectID: crypto.randomUUID(),
          module: moduleString,
          lang: "en",
        }),
        404
      );
    });

    it("returns 401 without access token", async () => {
      await expectStatus(
        schemaGenerate(api, "", {
          projectID: crypto.randomUUID(),
          module: moduleString,
          lang: "en",
        }),
        401
      );
    });
  },
  60000 * Object.keys(DEFAULT_WORKFLOWS).length
);
