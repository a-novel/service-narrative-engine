import { beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";
import {
  NarrativeEngineApi,
  moduleListVersions,
  projectDelete,
  projectInit,
  projectList,
  projectUpdate,
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

describe("projectInit", () => {
  it("creates a new project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "en",
      title: `Test Project ${Date.now()}`,
      workflow: [moduleString],
    });

    expect(project.id).toBeTruthy();
    expect(project.owner).toBeTruthy();
    expect(project.lang).toBe("en");
    expect(project.title).toContain("Test Project");
    expect(Array.isArray(project.workflow)).toBe(true);
    expect(project.workflow.length).toBe(1);
    expect(project.createdAt).toBeInstanceOf(Date);
    expect(project.updatedAt).toBeInstanceOf(Date);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("creates a project with French language", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "fr",
      title: `Projet Test ${Date.now()}`,
      workflow: [moduleString],
    });

    expect(project.lang).toBe("fr");

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 404 for non-existent module in workflow", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectInit(api, user.token.accessToken, {
        lang: "en",
        title: "Test Project",
        workflow: ["non-existent:module@v0.0.0"],
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectInit(api, "", {
        lang: "en",
        title: "Test Project",
        workflow: [moduleString],
      }),
      401
    );
  });
});

describe("projectList", () => {
  it("returns a list of projects", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    // Create a project first
    const project = await projectInit(api, user.token.accessToken, {
      lang: "en",
      title: `Test Project for List ${Date.now()}`,
      workflow: [moduleString],
    });

    const projects = await projectList(api, user.token.accessToken, {
      limit: 100,
      offset: 0,
    });

    expect(Array.isArray(projects)).toBe(true);
    expect(projects.length).toBeGreaterThan(0);
    expect(projects.some((p) => p.id === project.id)).toBe(true);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("respects limit parameter", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const projects = await projectList(api, user.token.accessToken, {
      limit: 1,
      offset: 0,
    });

    expect(projects.length).toBeLessThanOrEqual(1);
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectList(api, "", {
        limit: 10,
        offset: 0,
      }),
      401
    );
  });
});

describe("projectUpdate", () => {
  it("updates project title", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "en",
      title: "Original Title",
      workflow: [moduleString],
    });

    const updatedProject = await projectUpdate(api, user.token.accessToken, {
      id: project.id,
      title: "Updated Title",
      workflow: [moduleString],
    });

    expect(updatedProject.id).toBe(project.id);
    expect(updatedProject.title).toBe("Updated Title");
    expect(updatedProject.updatedAt.getTime()).toBeGreaterThanOrEqual(project.updatedAt.getTime());

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("updates project workflow", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "en",
      title: "Workflow Update Test",
      workflow: [moduleString],
    });

    // Update with same workflow (or a different valid one if available)
    const updatedProject = await projectUpdate(api, user.token.accessToken, {
      id: project.id,
      title: "Workflow Update Test",
      workflow: [moduleString],
    });

    expect(updatedProject.id).toBe(project.id);
    expect(Array.isArray(updatedProject.workflow)).toBe(true);

    // Cleanup
    await projectDelete(api, user.token.accessToken, { id: project.id });
  });

  it("returns 404 for non-existent project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectUpdate(api, user.token.accessToken, {
        id: crypto.randomUUID(),
        title: "Updated Title",
        workflow: [moduleString],
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectUpdate(api, "", {
        id: crypto.randomUUID(),
        title: "Updated Title",
        workflow: [moduleString],
      }),
      401
    );
  });
});

describe("projectDelete", () => {
  it("deletes a project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    const project = await projectInit(api, user.token.accessToken, {
      lang: "en",
      title: `Project to Delete ${Date.now()}`,
      workflow: [moduleString],
    });

    const deletedProject = await projectDelete(api, user.token.accessToken, {
      id: project.id,
    });

    expect(deletedProject.id).toBe(project.id);

    // Verify project is no longer in list
    const projects = await projectList(api, user.token.accessToken, {
      limit: 100,
      offset: 0,
    });

    expect(projects.some((p) => p.id === project.id)).toBe(false);
  });

  it("returns 404 for non-existent project", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectDelete(api, user.token.accessToken, {
        id: crypto.randomUUID(),
      }),
      404
    );
  });

  it("returns 401 without access token", async () => {
    const api = new NarrativeEngineApi(process.env.API_URL!);

    await expectStatus(
      projectDelete(api, "", {
        id: crypto.randomUUID(),
      }),
      401
    );
  });
});
