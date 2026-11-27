import { ABSENT_ID, accessToken, api, authApi } from "./fixtures";

import { describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { userAccessToken } from "@a-novel/service-authentication-rest-test";
import {
  DocumentRefusal,
  type InstanceDescriptor,
  Language,
  type Link,
  type Project,
  instanceContentUpdate,
  instanceDelete,
  instanceGet,
  instanceLinksList,
  instanceLinksUpdate,
  instancesCreate,
  projectCreate,
  projectDelete,
  projectEditsList,
} from "@a-novel/service-narrative-engine-rest";

/**
 * Creates a project of the session that holds two instances, and opens on the first. The test deletes it when it
 * finishes, whatever its outcome.
 */
async function seedProject(token = accessToken): Promise<{ project: Project; instances: InstanceDescriptor[] }> {
  const project = await projectCreate(api, token, { title: "Links", language: Language.En });
  onTestFinished(() =>
    projectDelete(api, token, project).then(
      () => undefined,
      () => undefined
    )
  );

  const instances = await instancesCreate(api, token, {
    projectID: project.id,
    instances: [
      { plugin: "scenes-composer", version: "1.2.0", name: "Composer", content: { beats: [] } },
      { plugin: "characters", version: "0.1.0", name: "Characters", content: [] },
    ],
  });

  return { project, instances };
}

/**
 * Names an instance in a links write. A write spreads this and never the instance itself, whose `updatedAt` is the
 * one of its content, not of its links.
 */
function of(instance: InstanceDescriptor): { projectID: string; id: string } {
  return { projectID: instance.projectID, id: instance.id };
}

/** A link to an item of an instance no project holds: the server never checks that it exists. */
const link: Link = {
  source: "scenes/a",
  destination: { id: "01920000-0000-7000-8000-000000000032", target: "characters/b" },
  relationship: "therefore",
};

describe("instanceLinksList", () => {
  it("returns an empty list and no updatedAt for each instance that never wrote its links", async () => {
    const { project, instances } = await seedProject();

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [
        { instanceID: instances[0].id, links: [], updatedAt: null },
        { instanceID: instances[1].id, links: [], updatedAt: null },
      ],
      total: 2,
    });
  });

  it("pages through the instances, oldest first, each page with their total", async () => {
    const { project, instances } = await seedProject();
    const written = await instanceLinksUpdate(api, accessToken, { ...of(instances[1]), links: [link] });

    await expect(instanceLinksList(api, accessToken, { projectID: project.id, limit: 1 })).resolves.toEqual({
      instances: [{ instanceID: instances[0].id, links: [], updatedAt: null }],
      total: 2,
    });
    await expect(instanceLinksList(api, accessToken, { projectID: project.id, limit: 1, offset: 1 })).resolves.toEqual({
      instances: [{ instanceID: instances[1].id, links: [link], updatedAt: written.updatedAt }],
      total: 2,
    });
    // A page past the end has no instance to carry the total.
    await expect(instanceLinksList(api, accessToken, { projectID: project.id, offset: 2 })).resolves.toEqual({
      instances: [],
      total: 0,
    });
  });

  it("returns 422 for a page larger than the service serves", async () => {
    const { project } = await seedProject();

    await expectStatus(instanceLinksList(api, accessToken, { projectID: project.id, limit: 501 }), 422);
  });

  it("returns 404 for the project of another user", async () => {
    const { project } = await seedProject();

    await expectStatus(instanceLinksList(api, await userAccessToken(authApi), { projectID: project.id }), 404);
    await expectStatus(instanceLinksList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });
});

describe("instanceLinksUpdate", () => {
  it("writes the first links of an instance without an updatedAt, then replaces them with it", async () => {
    const { project, instances } = await seedProject();

    const first = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link] });
    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [{ instanceID: instances[0].id, links: [link], updatedAt: first.updatedAt }, { links: [] }],
    });

    const next = { ...link, relationship: "but" };
    const second = await instanceLinksUpdate(api, accessToken, {
      ...of(instances[0]),
      links: [next, link],
      updatedAt: first.updatedAt,
    });
    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [{ instanceID: instances[0].id, links: [next, link], updatedAt: second.updatedAt }, { links: [] }],
    });
  });

  it("writes the links of two instances one after the other, and neither write is stale", async () => {
    const { project, instances } = await seedProject();

    // Each instance stamps its own links, so the write of one leaves the stamp the other names.
    const first = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link] });
    const other = await instanceLinksUpdate(api, accessToken, { ...of(instances[1]), links: [link] });
    const second = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [], ...first });

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [
        { instanceID: instances[0].id, links: [], updatedAt: second.updatedAt },
        { instanceID: instances[1].id, links: [link], updatedAt: other.updatedAt },
      ],
      total: 2,
    });
  });

  it("leaves a pending save of the content valid, and a content save leaves the links' updatedAt", async () => {
    const { instances } = await seedProject();
    const [composer] = instances;

    // The client read the instance, then its links are written before it saves the content it holds.
    const links = await instanceLinksUpdate(api, accessToken, { ...of(composer), links: [link] });
    const saved = await instanceContentUpdate(api, accessToken, { ...composer, content: { beats: [{ id: "a" }] } });
    expect(saved.updatedAt).not.toBe(composer.updatedAt);

    // The content write moved the instance's updatedAt alone: the links still answer to the one their write
    // returned.
    await instanceLinksUpdate(api, accessToken, { ...of(composer), links: [], ...links });
    await expect(instanceGet(api, accessToken, composer)).resolves.toMatchObject({
      updatedAt: saved.updatedAt,
      content: { beats: [{ id: "a" }] },
    });
  });

  it("clears the links with an empty list", async () => {
    const { project, instances } = await seedProject();
    const first = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link] });

    const cleared = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [], ...first });

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [
        { links: [], updatedAt: cleared.updatedAt },
        { links: [], updatedAt: null },
      ],
    });
  });

  it("stores a link from an item of an instance to another of its own items", async () => {
    const { project, instances } = await seedProject();
    const inner = { source: "scenes/a", destination: { id: instances[0].id, target: "scenes/b" }, relationship: "but" };

    await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [inner] });

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [{ links: [inner] }, { links: [] }],
    });
  });

  it("stamps the project's editedAt", async () => {
    const user = await userAccessToken(authApi);
    const { project, instances } = await seedProject(user);
    const { edits: before } = await projectEditsList(api, user);

    await instanceLinksUpdate(api, user, { ...of(instances[0]), links: [link] });

    const { edits } = await projectEditsList(api, user);
    expect(edits[project.id].getTime()).toBeGreaterThan(before[project.id].getTime());
  });

  it("returns 409 for a stale updatedAt, and changes nothing", async () => {
    const { project, instances } = await seedProject();
    const first = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link] });
    const second = await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [], ...first });

    // The first updatedAt names the links as they were before the second write.
    await expectStatus(instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link], ...first }), 409);
    // A write that names no updatedAt expects an instance that never wrote its links.
    await expectStatus(instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link] }), 409);
    // The updatedAt of the content names no write of the links.
    await expectStatus(
      instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [link], updatedAt: instances[0].updatedAt }),
      409
    );

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [
        { links: [], updatedAt: second.updatedAt },
        { links: [], updatedAt: null },
      ],
    });
  });

  it("returns 409 for a first write that names an updatedAt", async () => {
    const { project, instances } = await seedProject();

    // An instance that never wrote its links holds no updatedAt for them, so no instant names them.
    await expectStatus(
      instanceLinksUpdate(api, accessToken, {
        ...instances[0],
        links: [link],
        updatedAt: "2021-01-01T00:00:00.000001Z",
      }),
      409
    );

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [
        { links: [], updatedAt: null },
        { links: [], updatedAt: null },
      ],
    });
  });

  it("returns 422 with the limit for links the server refuses", async () => {
    const { project, instances } = await seedProject();
    const refuse = (links: Link[]) => instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links });

    // By default a target holds 200 characters, a relationship 1 KiB, and an instance 20,000 links.
    await expect(refuse([link, { ...link, source: "é".repeat(201) }])).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { "links[1].source": DocumentRefusal.Max } },
    });
    await expect(
      refuse([{ ...link, destination: { ...link.destination, target: "é".repeat(201) } }])
    ).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { "links[0].destination.target": DocumentRefusal.Max } },
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

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [
        { links: [], updatedAt: null },
        { links: [], updatedAt: null },
      ],
    });
  });

  it("accepts targets of 200 characters and a relationship of 1 KiB", async () => {
    const { project, instances } = await seedProject();
    const widest = {
      source: "é".repeat(200),
      destination: { ...link.destination, target: "é".repeat(200) },
      relationship: "a".repeat(1024),
    };

    await instanceLinksUpdate(api, accessToken, { ...of(instances[0]), links: [widest] });

    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({
      instances: [{ links: [widest] }, { links: [] }],
    });
  });

  it("returns 400 for a link whose destination ID is no UUID", async () => {
    const { instances } = await seedProject();

    await expectStatus(
      instanceLinksUpdate(api, accessToken, {
        ...instances[0],
        links: [{ ...link, destination: { id: "not-a-uuid", target: "a" } }],
      }),
      400
    );
  });

  it("returns 404 for an instance the project lacks, and for the project of another user", async () => {
    const { project, instances } = await seedProject();

    await expectStatus(
      instanceLinksUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, links: [link] }),
      404
    );
    await expectStatus(
      instanceLinksUpdate(api, await userAccessToken(authApi), { ...of(instances[0]), links: [link] }),
      404
    );
  });
});

describe("instanceDelete, of an instance that stores links", () => {
  it("deletes the links the instance stores, and leaves the links that point at it", async () => {
    const { project, instances } = await seedProject();
    const [composer, characters] = instances;
    // The composer, the default instance, points at the characters, which store a link of their own.
    const pointing = { ...link, destination: { id: characters.id, target: "characters/b" } };
    const kept = await instanceLinksUpdate(api, accessToken, { ...of(composer), links: [pointing] });
    await instanceLinksUpdate(api, accessToken, { ...of(characters), links: [link] });

    await instanceDelete(api, accessToken, characters);

    // The link that points at the deleted instance stays where it is stored: the client renders it as broken.
    await expect(instanceLinksList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [{ instanceID: composer.id, links: [pointing], updatedAt: kept.updatedAt }],
      total: 1,
    });
  });
});
