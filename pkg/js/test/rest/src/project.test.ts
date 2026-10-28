import { anonymousAccessToken, superAdminAccessToken } from "./token";

import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  Language,
  LayerKind,
  NarrativeEngineApi,
  projectCreate,
  projectDelete,
  projectGet,
  projectList,
  projectUpdate,
} from "@a-novel/service-narrative-engine-rest";

const api = new NarrativeEngineApi(process.env.REST_URL!);
const accessToken = await superAdminAccessToken();

const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

describe("projectCreate", () => {
  it("creates a project with its required layers", async () => {
    const project = await projectCreate(api, accessToken, {
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: Language.Fr,
    });

    expect(project).toMatchObject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: "fr",
    });
    expect(project.layers.map((layer) => [layer.position, layer.kind])).toEqual([[0, LayerKind.Manuscript]]);

    await projectDelete(api, accessToken, project);
  });

  it("names the invalid field of a blank title", async () => {
    await expect(projectCreate(api, accessToken, { title: "   ", language: Language.En })).rejects.toThrow(
      '"invalidFields":{"title":"notblank"}'
    );
  });

  it("returns 401 without a session", async () => {
    await expectStatus(projectCreate(api, "", { title: "Clear Hours", language: Language.En }), 401);
  });

  it("returns 403 for an anonymous session", async () => {
    const anonymous = await anonymousAccessToken();
    await expectStatus(projectCreate(api, anonymous, { title: "Clear Hours", language: Language.En }), 403);
  });
});

describe("projectGet", () => {
  it("returns the project", async () => {
    const created = await projectCreate(api, accessToken, {
      title: "Clear Hours",
      premise: "Marie waits.",
      language: Language.En,
    });

    const project = await projectGet(api, accessToken, { id: created.id });
    expect(project).toEqual({
      id: created.id,
      title: "Clear Hours",
      premise: "Marie waits.",
      language: Language.En,
      createdAt: created.createdAt,
      updatedAt: created.updatedAt,
    });

    await projectDelete(api, accessToken, project);
  });

  it("returns 400 for a malformed ID", async () => {
    await expectStatus(projectGet(api, accessToken, { id: "not-a-uuid" }), 400);
  });

  it("returns 404 for an absent project", async () => {
    await expectStatus(projectGet(api, accessToken, { id: ABSENT_ID }), 404);
  });
});

describe("projectList", () => {
  it("lists projects newest first, a page at a time, with their total", async () => {
    const older = await projectCreate(api, accessToken, { title: "Older", language: Language.En });
    const newer = await projectCreate(api, accessToken, { title: "Newer", language: Language.En });

    const ids = (await projectList(api, accessToken)).projects.map((project) => project.id);
    expect(ids.indexOf(newer.id)).toBeGreaterThanOrEqual(0);
    expect(ids.indexOf(newer.id)).toBeLessThan(ids.indexOf(older.id));

    // Other test files create projects meanwhile, so the page after the newer project still holds the older one.
    const offset = ids.indexOf(newer.id) + 1;
    const nextPage = (await projectList(api, accessToken, { offset, limit: 100 })).projects.map(
      (project) => project.id
    );
    expect(nextPage).toContain(older.id);

    const firstPage = await projectList(api, accessToken, { limit: 1 });
    expect(firstPage.projects).toHaveLength(1);
    expect(firstPage.total).toBeGreaterThanOrEqual(2);

    await projectDelete(api, accessToken, older);
    await projectDelete(api, accessToken, newer);
  });

  it("returns 400 for a page over 100 projects", async () => {
    await expectStatus(projectList(api, accessToken, { limit: 101 }), 400);
  });

  it("returns 400 for a negative offset", async () => {
    await expectStatus(projectList(api, accessToken, { offset: -1 }), 400);
  });
});

describe("projectUpdate", () => {
  it("replaces the fields", async () => {
    const created = await projectCreate(api, accessToken, { title: "Draft", language: Language.En });

    const project = await projectUpdate(api, accessToken, {
      id: created.id,
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: Language.Fr,
    });
    expect(project).toMatchObject({
      title: "Les Heures claires",
      premise: "Marie attend.",
      language: "fr",
    });

    await projectDelete(api, accessToken, project);
  });

  it("returns 404 for an absent project", async () => {
    const request = { id: ABSENT_ID, title: "Renamed", language: Language.En };
    await expectStatus(projectUpdate(api, accessToken, request), 404);
  });
});

describe("projectDelete", () => {
  it("deletes the project", async () => {
    const created = await projectCreate(api, accessToken, { title: "Doomed", language: Language.En });

    await expect(projectDelete(api, accessToken, created)).resolves.toMatchObject({ id: created.id });
    await expectStatus(projectGet(api, accessToken, { id: created.id }), 404);
  });
});
