import { ABSENT_ID, accessToken, api, authApi } from "./fixtures";

import { describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { userAccessToken } from "@a-novel/service-authentication-rest-test";
import {
  DocumentRefusal,
  Language,
  type Link,
  type Project,
  projectCreate,
  projectDelete,
  projectEditsList,
  projectLinksGet,
  projectLinksUpdate,
} from "@a-novel/service-narrative-engine-rest";

/** Creates a project of the session the test deletes when it finishes, whatever its outcome. */
async function createProject(token = accessToken): Promise<Project> {
  const project = await projectCreate(api, token, { title: "Links", language: Language.En });
  onTestFinished(() =>
    projectDelete(api, token, project).then(
      () => undefined,
      () => undefined
    )
  );

  return project;
}

/** A link between two instances no project holds: the server never checks that they exist. */
const link: Link = {
  source: { id: "01920000-0000-7000-8000-000000000031", target: "scenes/a" },
  destination: { id: "01920000-0000-7000-8000-000000000032", target: "characters/b" },
  relationship: "therefore",
};

describe("projectLinksGet", () => {
  it("returns an empty list and no updatedAt for a project that never wrote its links", async () => {
    const project = await createProject();

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [],
      updatedAt: null,
    });
  });

  it("returns 404 for the project of another user", async () => {
    const project = await createProject();

    await expectStatus(projectLinksGet(api, await userAccessToken(authApi), { projectID: project.id }), 404);
    await expectStatus(projectLinksGet(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});

describe("projectLinksUpdate", () => {
  it("writes the first links of a project without an updatedAt, then replaces them with it", async () => {
    const project = await createProject();

    const first = await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [link] });
    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [link],
      updatedAt: first.updatedAt,
    });

    const next = { ...link, relationship: "but" };
    const second = await projectLinksUpdate(api, accessToken, {
      projectID: project.id,
      links: [next, link],
      updatedAt: first.updatedAt,
    });
    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [next, link],
      updatedAt: second.updatedAt,
    });
  });

  it("clears the links with an empty list", async () => {
    const project = await createProject();
    const first = await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [link] });

    const cleared = await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [], ...first });

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [],
      updatedAt: cleared.updatedAt,
    });
  });

  it("stamps the project's editedAt", async () => {
    const user = await userAccessToken(authApi);
    const project = await createProject(user);

    await projectLinksUpdate(api, user, { projectID: project.id, links: [link] });

    const { edits } = await projectEditsList(api, user);
    expect(edits[project.id].getTime()).toBeGreaterThan(project.editedAt.getTime());
  });

  it("returns 409 for a stale updatedAt, and changes nothing", async () => {
    const project = await createProject();
    const first = await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [link] });
    const second = await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [], ...first });

    // The first updatedAt names the links as they were before the second write.
    await expectStatus(projectLinksUpdate(api, accessToken, { projectID: project.id, links: [link], ...first }), 409);
    // A write that names no updatedAt expects a project that never wrote its links.
    await expectStatus(projectLinksUpdate(api, accessToken, { projectID: project.id, links: [link] }), 409);

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [],
      updatedAt: second.updatedAt,
    });
  });

  it("writes the first links of a project whatever updatedAt the request names", async () => {
    const project = await createProject();

    // No client reads an instant from links a project never wrote, so the write has nothing to be stale against.
    const written = await projectLinksUpdate(api, accessToken, {
      projectID: project.id,
      links: [link],
      updatedAt: "2021-01-01T00:00:00.000001Z",
    });

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [link],
      updatedAt: written.updatedAt,
    });
  });

  it("returns 422 with the limit for links the server refuses", async () => {
    const project = await createProject();
    const refuse = (links: Link[]) => projectLinksUpdate(api, accessToken, { projectID: project.id, links });

    // By default a target holds 200 characters, a relationship 1 KiB, and a project 20,000 links.
    await expect(
      refuse([link, { ...link, destination: { ...link.destination, target: "é".repeat(201) } }])
    ).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { "links[1].destination.target": DocumentRefusal.Max } },
    });
    await expect(refuse([{ ...link, relationship: "a".repeat(1025) }])).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { "links[0].relationship": DocumentRefusal.MaxBytes } },
    });
    await expect(refuse(Array.from({ length: 20001 }, () => link))).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { links: DocumentRefusal.Max } },
    });
    await expect(refuse([{ ...link, relationship: "a\u0000b" }])).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { links: DocumentRefusal.NoNul } },
    });

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toEqual({
      links: [],
      updatedAt: null,
    });
  });

  it("accepts a target of 200 characters and a relationship of 1 KiB", async () => {
    const project = await createProject();
    const widest = {
      source: { ...link.source, target: "é".repeat(200) },
      destination: link.destination,
      relationship: "a".repeat(1024),
    };

    await projectLinksUpdate(api, accessToken, { projectID: project.id, links: [widest] });

    await expect(projectLinksGet(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      links: [widest],
    });
  });

  it("returns 400 for a link whose ID is no UUID", async () => {
    const project = await createProject();

    await expectStatus(
      projectLinksUpdate(api, accessToken, {
        projectID: project.id,
        links: [{ ...link, source: { id: "not-a-uuid", target: "a" } }],
      }),
      400
    );
  });

  it("returns 404 for the project of another user", async () => {
    const project = await createProject();

    await expectStatus(
      projectLinksUpdate(api, await userAccessToken(authApi), { projectID: project.id, links: [link] }),
      404
    );
  });
});
