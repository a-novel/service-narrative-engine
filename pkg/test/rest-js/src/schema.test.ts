import { beforeAll, describe, expect, it } from "vitest";

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

let user: Awaited<ReturnType<typeof registerUser>>;
let version: string;
let preversion: string | undefined;
let moduleString: string;

const TEST_MODULE_NAMESPACE = "agora";
const TEST_MODULE_ID = "idea";

beforeAll(async () => {
  const authApi = new AuthenticationApi(process.env.AUTH_API_URL!);
  const api = new NarrativeEngineApi(process.env.API_URL!);

  const preRegister = await preRegisterUser(authApi, process.env.MAIL_TEST_HOST!);
  user = await registerUser(authApi, preRegister);

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

async function createTestProject(api: NarrativeEngineApi, accessToken: string) {
  return await projectInit(api, accessToken, {
    lang: "en",
    title: `Schema Test Project ${Date.now()}`,
    workflow: [moduleString],
  });
}

describe("schemaCreate", () => {
  it("creates a new schema", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

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

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("creates schema with AI source", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

    const schema = await schemaCreate(api, user.token.accessToken, {
      id: crypto.randomUUID(),
      projectID: project.id,
      module: moduleString,
      source: "AI",
      data: { generated: true },
    });

    expect(schema.source).toBe("AI");

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 404 for non-existent project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

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
    const api = new NarrativeEngineApi(process.env.API_URL!);

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
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

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

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("selects a schema by module", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

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
    expect(schema.module).toContain(`${TEST_MODULE_NAMESPACE}:${TEST_MODULE_ID}@v`);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 404 for non-existent schema", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

    await expectStatus(
      schemaSelect(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        projectID: project.id,
      }),
      404
    );

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
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
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

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

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 404 for non-existent schema", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      schemaRewrite(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        data: { test: "data" },
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

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
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

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

    const versions = await schemaListVersions(api, user.token.accessToken, {
      projectID: project.id,
      moduleID: TEST_MODULE_ID,
      moduleNamespace: TEST_MODULE_NAMESPACE,
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBeGreaterThan(0);

    const version = versions[0];
    expect(version.id).toBeTruthy();
    expect(version.createdAt).toBeInstanceOf(Date);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns empty array for non-existent module", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

    const versions = await schemaListVersions(api, user.token.accessToken, {
      projectID: project.id,
      moduleID: "non-existent",
      moduleNamespace: "non-existent",
      limit: 10,
      offset: 0,
    });

    expect(Array.isArray(versions)).toBe(true);
    expect(versions.length).toBe(0);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

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

describe("schemaGenerate", () => {
  // AI generation tests have longer timeouts due to LLM API latency
  it("generates a schema using AI", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);
    const project = await createTestProject(api, user.token.accessToken);

    const schema = await schemaGenerate(api, user.token.accessToken, {
      projectID: project.id,
      module: moduleString,
      lang: "en",
    });

    expect(schema.id).toBeTruthy();
    expect(schema.projectID).toBe(project.id);
    // Schema returns the stored module version which may differ from the request.
    expect(schema.module).toContain(`${TEST_MODULE_NAMESPACE}:${TEST_MODULE_ID}@v`);
    expect(schema.source).toBe("AI");
    expect(schema.data).toBeTruthy();
    expect(schema.createdAt).toBeInstanceOf(Date);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  }, 60000);

  it("generates a schema in French", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "fr",
      title: `Schema Generate FR Test ${Date.now()}`,
      workflow: [moduleString],
    });

    const schema = await schemaGenerate(api, user.token.accessToken, {
      projectID: project.id,
      module: moduleString,
      lang: "fr",
    });

    expect(schema.source).toBe("AI");

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  }, 60000);

  it("returns 404 for non-existent project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

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
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      schemaGenerate(api, "", {
        projectID: crypto.randomUUID(),
        module: moduleString,
        lang: "en",
      }),
      401
    );
  });
});
