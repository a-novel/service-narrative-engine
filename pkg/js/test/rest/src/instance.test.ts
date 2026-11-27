import { ABSENT_ID, accessToken, api, authApi } from "./fixtures";

import { describe, expect, it, onTestFinished } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { userAccessToken } from "@a-novel/service-authentication-rest-test";
import {
  DocumentRefusal,
  type InstanceDescriptor,
  InstanceNameSchema,
  InstanceVersionSchema,
  type InstancesCreateItem,
  Language,
  type Project,
  instanceContentUpdate,
  instanceCreate,
  instanceDelete,
  instanceGet,
  instanceList,
  instanceUpdate,
  instancesCreate,
  projectCreate,
  projectDelete,
  projectEditsList,
  projectGet,
} from "@a-novel/service-narrative-engine-rest";

/** Creates a project of the session the test deletes when it finishes, whatever its outcome. */
async function createProject(token = accessToken): Promise<Project> {
  const project = await projectCreate(api, token, { title: "Instances", language: Language.En });
  onTestFinished(() =>
    projectDelete(api, token, project).then(
      () => undefined,
      () => undefined
    )
  );

  return project;
}

/** An instance of the composer, whose content is small. */
const composer: InstancesCreateItem = {
  plugin: "scenes-composer",
  version: "1.2.0",
  name: "Composer",
  content: { beats: [{ id: "a", scenes: [] }] },
};

/** Creates a project that holds the composer and a second instance, and opens on the composer. */
async function seedProject(token = accessToken): Promise<{ project: Project; instances: InstanceDescriptor[] }> {
  const project = await createProject(token);
  const instances = await instancesCreate(api, token, {
    projectID: project.id,
    instances: [composer, { plugin: "characters", version: "0.1.0-dev.1", name: "Characters", content: [] }],
  });

  return { project, instances };
}

/** Nests `depth` arrays inside one another. */
function nested(depth: number): unknown {
  let value: unknown = [];
  for (let level = 1; level < depth; level++) value = [value];

  return value;
}

describe("InstanceNameSchema", () => {
  it("accepts the names the server accepts", () => {
    for (const name of ["", "Composer", " a ", "é".repeat(80), "😀".repeat(80)]) {
      expect(InstanceNameSchema.safeParse(name).success).toBe(true);
    }
  });

  it("refuses the names the server refuses", () => {
    for (const name of ["   ", "é".repeat(81)]) expect(InstanceNameSchema.safeParse(name).success).toBe(false);
  });
});

describe("instanceCreate", () => {
  it("creates an instance, which becomes the default of a project that holds none", async () => {
    const project = await createProject();
    expect(project.defaultInstanceID).toBeUndefined();

    const instance = await instanceCreate(api, accessToken, { projectID: project.id, ...composer });
    expect(instance).toMatchObject({
      projectID: project.id,
      plugin: "scenes-composer",
      version: "1.2.0",
      name: "Composer",
    });
    // The instant keeps the microseconds the server stores, which a Date would drop.
    expect(instance.updatedAt).toMatch(/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{1,6})?Z$/);

    await expect(projectGet(api, accessToken, project)).resolves.toMatchObject({ defaultInstanceID: instance.id });
  });

  it("leaves the default of a project that has one", async () => {
    const { project, instances } = await seedProject();

    await instanceCreate(api, accessToken, { projectID: project.id, ...composer, name: "Second composer" });

    await expect(projectGet(api, accessToken, project)).resolves.toMatchObject({
      defaultInstanceID: instances[0].id,
    });
  });

  it("stores any JSON value as the content", async () => {
    const project = await createProject();

    for (const content of [null, 12, "text", [], { nested: { value: true } }]) {
      const created = await instanceCreate(api, accessToken, { projectID: project.id, ...composer, content });
      const instance = await instanceGet(api, accessToken, created);
      expect(instance.content).toEqual(content);
    }
  });

  it("stamps the project's editedAt, and its updatedAt only when the instance becomes the default", async () => {
    const user = await userAccessToken(authApi);
    const project = await createProject(user);

    // The first instance becomes the default, which is a field of the project.
    await instanceCreate(api, user, { projectID: project.id, ...composer });
    const opened = await projectGet(api, user, project);
    expect(opened.updatedAt.getTime()).toBeGreaterThan(project.updatedAt.getTime());

    await instanceCreate(api, user, { projectID: project.id, ...composer, name: "Second composer" });

    const { edits } = await projectEditsList(api, user);
    expect(edits[project.id].getTime()).toBeGreaterThan(opened.editedAt.getTime());
    await expect(projectGet(api, user, project)).resolves.toMatchObject({ updatedAt: opened.updatedAt });
  });

  it("returns 422 for a plugin name or a version the registry could not hold", async () => {
    const project = await createProject();

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, plugin: "Scenes Composer" })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { plugin: "dns_rfc1035_label" } } });
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, version: "^1.2.0" })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { version: "pluginversion" } } });
  });

  it("stores a version of 256 characters, and returns 422 for one more", async () => {
    const project = await createProject();
    const version = "1.2.0-" + "a".repeat(250);
    expect(InstanceVersionSchema.safeParse(version).success).toBe(true);
    expect(InstanceVersionSchema.safeParse(version + "a").success).toBe(false);

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, version })
    ).resolves.toMatchObject({ version });
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, version: version + "a" })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { version: "max" } } });
  });

  it("creates an instance without a name", async () => {
    const project = await createProject();
    const { plugin, version, content } = composer;

    const unnamed = await instanceCreate(api, accessToken, { projectID: project.id, plugin, version, content });
    expect(unnamed.name).toBe("");

    const empty = await instanceCreate(api, accessToken, { projectID: project.id, ...composer, name: "" });
    expect(empty.name).toBe("");
  });

  it("returns 422 for a name of only whitespace", async () => {
    const project = await createProject();

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, name: "  " })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { name: "notblank" } } });
  });

  it("returns 422 with the limit for a content over its size", async () => {
    const project = await createProject();
    // The server caps a content at 4 MiB by default.
    const content = "a".repeat(4 * 1024 * 1024);

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.MaxBytes } } });
  });

  it("stores a content nested as deep as the limit, and returns 422 one level past it", async () => {
    const project = await createProject();
    // The server caps the depth of a document at 32 by default.
    await instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: nested(32) });

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: nested(33) })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.MaxDepth } } });
  });

  it("returns 422 for a content holding a NUL character", async () => {
    const project = await createProject();

    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: { title: "a\u0000b" } })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoNul } } });
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: { "a\u0000b": 1 } })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoNul } } });
  });

  it("stores an emoji, and returns 422 for half of one", async () => {
    const project = await createProject();
    const content = { title: "Les Heures claires 😀" };

    const created = await instanceCreate(api, accessToken, { projectID: project.id, ...composer, content });
    await expect(instanceGet(api, accessToken, created)).resolves.toMatchObject({ content });

    // A string cut inside an emoji keeps half of its surrogate pair, which JSON spells as an escape.
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: { title: "😀".slice(0, 1) } })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoLoneSurrogate } } });
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content: { ["😀".slice(1)]: 1 } })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoLoneSurrogate } } });
  });

  it("counts a number written in full, as the server stores it", async () => {
    const project = await createProject();

    const created = await instanceCreate(api, accessToken, {
      projectID: project.id,
      ...composer,
      content: [Number.MAX_VALUE, Number.MIN_VALUE, 1e-7],
    });
    await expect(instanceGet(api, accessToken, created)).resolves.toMatchObject({
      content: [Number.MAX_VALUE, Number.MIN_VALUE, 1e-7],
    });

    // 140 KB of JSON the server would store as 6 MB: each 1e+300 is 301 digits written in full.
    const content = Array.from({ length: 20_000 }, () => 1e300);
    await expect(
      instanceCreate(api, accessToken, { projectID: project.id, ...composer, content })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.MaxBytes } } });
  });

  it("returns 422 for a number the server cannot store", async () => {
    const project = await createProject();
    // No JavaScript number is this large, so the client cannot send it: the request is written by hand.
    const response = await api.fetchResponse("/v1/instance", {
      method: "POST",
      accessToken,
      headers: { "Content-Type": "application/json" },
      body: `{"projectID":"${project.id}","plugin":"scenes-composer","version":"1.2.0","content":[1e999999]}`,
    });

    expect(response.status).toBe(422);
    await expect(response.json()).resolves.toMatchObject({
      tags: { invalidFields: { content: DocumentRefusal.MaxDigits } },
    });
  });

  it("returns 404 for the project of another user", async () => {
    const project = await createProject();
    const other = await userAccessToken(authApi);

    await expectStatus(instanceCreate(api, other, { projectID: project.id, ...composer }), 404);
    await expectStatus(instanceCreate(api, accessToken, { projectID: ABSENT_ID, ...composer }), 404);
  });
});

describe("instancesCreate", () => {
  it("creates the instances in the order of the request, and opens the project on the first", async () => {
    const { project, instances } = await seedProject();

    expect(instances.map((instance) => instance.name)).toEqual(["Composer", "Characters"]);
    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toEqual({ instances, total: 2 });
    await expect(projectGet(api, accessToken, project)).resolves.toMatchObject({
      defaultInstanceID: instances[0].id,
    });
  });

  it("creates none of the instances when one is refused, and names it by its place", async () => {
    const project = await createProject();

    await expect(
      instancesCreate(api, accessToken, {
        projectID: project.id,
        instances: [composer, { ...composer, content: nested(33) }],
      })
    ).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { "instances[1].content": DocumentRefusal.MaxDepth } },
    });

    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [],
      total: 0,
    });
  });

  it("fills a project, then returns 409 with the limit", async () => {
    const project = await createProject();
    // The server caps a project at 50 instances by default.
    const batch = Array.from({ length: 50 }, (_, index) => ({ ...composer, name: `Composer ${index}` }));

    await expect(instancesCreate(api, accessToken, { projectID: project.id, instances: batch })).resolves.toHaveLength(
      50
    );

    await expect(instanceCreate(api, accessToken, { projectID: project.id, ...composer })).rejects.toMatchObject({
      status: 409,
      tags: { limit: "instances" },
    });
    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({ total: 50 });
  });

  it("returns 413 for a batch larger than a project stores", async () => {
    const project = await createProject();
    // The server takes 20 MiB by default, above the 16 MiB a project stores. Each content here passes its own
    // 4 MiB cap, and the six pass that limit by under 256 KiB, which the server still reads before it answers.
    const sizes = [3.5, 3.5, 3.5, 3.5, 3.5, 2.5625].map((mebibytes) => mebibytes * 1024 * 1024);
    const batch = sizes.map((size) => ({ ...composer, content: "a".repeat(size) }));

    await expectStatus(instancesCreate(api, accessToken, { projectID: project.id, instances: batch }), 413);
    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toMatchObject({ total: 0 });
  });

  it("returns 422 for a batch a project cannot hold", async () => {
    const project = await createProject();
    const batch = Array.from({ length: 51 }, () => composer);

    await expect(instancesCreate(api, accessToken, { projectID: project.id, instances: batch })).rejects.toMatchObject({
      status: 422,
      tags: { invalidFields: { instances: DocumentRefusal.Max } },
    });
  });

  it("returns 404 for the project of another user", async () => {
    const project = await createProject();

    await expectStatus(
      instancesCreate(api, await userAccessToken(authApi), { projectID: project.id, instances: [composer] }),
      404
    );
  });
});

describe("instanceList", () => {
  it("returns an empty page for a project without an instance", async () => {
    const project = await createProject();

    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [],
      total: 0,
    });
  });

  it("pages through the instances, oldest first, each page with their total", async () => {
    const { project, instances } = await seedProject();

    await expect(instanceList(api, accessToken, { projectID: project.id, limit: 1 })).resolves.toEqual({
      instances: [instances[0]],
      total: 2,
    });
    await expect(instanceList(api, accessToken, { projectID: project.id, limit: 1, offset: 1 })).resolves.toEqual({
      instances: [instances[1]],
      total: 2,
    });
    // A page past the end has no instance to carry the total.
    await expect(instanceList(api, accessToken, { projectID: project.id, offset: 2 })).resolves.toEqual({
      instances: [],
      total: 0,
    });
  });

  it("returns 422 for a page larger than the service serves", async () => {
    const project = await createProject();

    await expectStatus(instanceList(api, accessToken, { projectID: project.id, limit: 501 }), 422);
  });

  it("returns 404 for the project of another user", async () => {
    const { project } = await seedProject();

    await expectStatus(instanceList(api, await userAccessToken(authApi), { projectID: project.id }), 404);
    await expectStatus(instanceList(api, accessToken, { projectID: ABSENT_ID }), 404);
  });

  it("returns 400 for a malformed project ID", async () => {
    await expectStatus(instanceList(api, accessToken, { projectID: "not-a-uuid" }), 400);
  });
});

describe("instanceGet", () => {
  it("returns the instance with its content", async () => {
    const { instances } = await seedProject();

    await expect(instanceGet(api, accessToken, instances[0])).resolves.toEqual({
      ...instances[0],
      content: composer.content,
    });
  });

  it("returns 404 for an instance the project lacks, and for the project of another user", async () => {
    const { project, instances } = await seedProject();
    const other = await userAccessToken(authApi);
    const { project: otherProject } = await seedProject(other);

    await expectStatus(instanceGet(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
    await expectStatus(instanceGet(api, other, instances[0]), 404);
    // An instance answers only through the project that holds it.
    await expectStatus(instanceGet(api, other, { projectID: otherProject.id, id: instances[0].id }), 404);
  });
});

describe("instanceUpdate", () => {
  it("renames the instance, and leaves its updatedAt", async () => {
    const { instances } = await seedProject();

    const renamed = await instanceUpdate(api, accessToken, { ...instances[0], name: "Outline" });
    expect(renamed).toEqual({ ...instances[0], name: "Outline" });

    // The content write still names the updatedAt read before the rename.
    await expect(
      instanceContentUpdate(api, accessToken, { ...instances[0], content: { beats: [] } })
    ).resolves.toMatchObject({ name: "Outline" });
  });

  it("clears the name with an empty one", async () => {
    const { instances } = await seedProject();

    await expect(instanceUpdate(api, accessToken, { ...instances[0], name: "" })).resolves.toEqual({
      ...instances[0],
      name: "",
    });
  });

  it("returns 422 for a name of only whitespace", async () => {
    const { instances } = await seedProject();

    await expectStatus(instanceUpdate(api, accessToken, { ...instances[0], name: " " }), 422);
  });

  it("returns 404 for an instance the project lacks, and for the project of another user", async () => {
    const { project, instances } = await seedProject();

    await expectStatus(
      instanceUpdate(api, accessToken, { projectID: project.id, id: ABSENT_ID, name: "Outline" }),
      404
    );
    await expectStatus(instanceUpdate(api, await userAccessToken(authApi), { ...instances[0], name: "Outline" }), 404);
  });
});

describe("instanceContentUpdate", () => {
  it("replaces the version and the content", async () => {
    const { instances } = await seedProject();

    const written = await instanceContentUpdate(api, accessToken, {
      ...instances[0],
      version: "2.0.0-dev.1",
      content: { beats: [], migrated: true },
    });
    expect(written).toEqual({ ...instances[0], version: "2.0.0-dev.1", updatedAt: written.updatedAt });
    expect(written.updatedAt).not.toBe(instances[0].updatedAt);

    await expect(instanceGet(api, accessToken, written)).resolves.toEqual({
      ...written,
      content: { beats: [], migrated: true },
    });
  });

  it("returns 409 for a stale updatedAt, and changes nothing", async () => {
    const { instances } = await seedProject();
    const written = await instanceContentUpdate(api, accessToken, { ...instances[0], content: { saved: 1 } });

    // The first updatedAt names the instance as it was before that write.
    await expectStatus(instanceContentUpdate(api, accessToken, { ...instances[0], content: { saved: 2 } }), 409);

    await expect(instanceGet(api, accessToken, written)).resolves.toEqual({
      ...written,
      content: { saved: 1 },
    });
  });

  it("returns 409 for an updatedAt a Date rounded", async () => {
    const { instances } = await seedProject();
    let current = instances[0];

    // One write in a thousand lands on a whole millisecond, which a Date names exactly: write until one does not.
    while (!/\.\d{4,6}Z$/.test(current.updatedAt)) {
      current = await instanceContentUpdate(api, accessToken, { ...current, content: {} });
    }

    await expectStatus(
      instanceContentUpdate(api, accessToken, {
        ...current,
        content: { saved: true },
        updatedAt: new Date(current.updatedAt).toISOString(),
      }),
      409
    );
  });

  it("returns 422 with the limit for a content the server refuses", async () => {
    const { instances } = await seedProject();
    const tooLarge = "a".repeat(4 * 1024 * 1024);

    await expect(instanceContentUpdate(api, accessToken, { ...instances[0], content: tooLarge })).rejects.toMatchObject(
      { status: 422, tags: { invalidFields: { content: DocumentRefusal.MaxBytes } } }
    );
    await expect(
      instanceContentUpdate(api, accessToken, { ...instances[0], content: nested(33) })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.MaxDepth } } });
    await expect(
      instanceContentUpdate(api, accessToken, { ...instances[0], content: ["a\u0000"] })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoNul } } });
    await expect(
      instanceContentUpdate(api, accessToken, { ...instances[0], content: ["😀".slice(0, 1)] })
    ).rejects.toMatchObject({ status: 422, tags: { invalidFields: { content: DocumentRefusal.NoLoneSurrogate } } });

    // Every refused write left the instance as it was.
    await expect(instanceGet(api, accessToken, instances[0])).resolves.toMatchObject({
      updatedAt: instances[0].updatedAt,
      content: composer.content,
    });
  });

  it("returns 404 for an instance the project lacks, and for the project of another user", async () => {
    const { project, instances } = await seedProject();

    await expectStatus(
      instanceContentUpdate(api, accessToken, { ...instances[0], projectID: project.id, id: ABSENT_ID, content: {} }),
      404
    );
    await expectStatus(
      instanceContentUpdate(api, await userAccessToken(authApi), { ...instances[0], content: {} }),
      404
    );
  });
});

describe("instanceDelete", () => {
  it("deletes an instance that is not the default", async () => {
    const { project, instances } = await seedProject();

    await expect(instanceDelete(api, accessToken, instances[1])).resolves.toEqual(instances[1]);

    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toEqual({
      instances: [instances[0]],
      total: 1,
    });
    await expectStatus(instanceGet(api, accessToken, instances[1]), 404);
  });

  it("returns 409 for the default instance, which stays", async () => {
    const { project, instances } = await seedProject();

    await expectStatus(instanceDelete(api, accessToken, instances[0]), 409);

    await expect(instanceList(api, accessToken, { projectID: project.id })).resolves.toEqual({ instances, total: 2 });
  });

  it("returns 404 for an instance the project lacks, and for the project of another user", async () => {
    const { project, instances } = await seedProject();

    await expectStatus(instanceDelete(api, accessToken, { projectID: project.id, id: ABSENT_ID }), 404);
    await expectStatus(instanceDelete(api, await userAccessToken(authApi), instances[1]), 404);
  });
});
