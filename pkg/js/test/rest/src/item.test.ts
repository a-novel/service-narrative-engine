import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { TemplateApi, itemCreate, itemDelete, itemGet, itemList, itemUpdate } from "@a-novel/service-template-rest";

describe("itemCreate", () => {
  it("creates a new item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const item = await itemCreate(api, "test item", "test description");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("test item");
    expect(item.description).toBe("test description");
    expect(item.createdAt).toBeTruthy();
    expect(item.updatedAt).toBeTruthy();

    await itemDelete(api, item.id);
  });

  it("creates an item without description", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const item = await itemCreate(api, "no description item");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("no description item");

    await itemDelete(api, item.id);
  });

  it("returns 400 for empty name", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemCreate(api, ""), 400);
  });
});

describe("itemGet", () => {
  it("retrieves an existing item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const created = await itemCreate(api, "item to get");

    const item = await itemGet(api, created.id);
    expect(item.id).toBe(created.id);
    expect(item.name).toBe("item to get");

    await itemDelete(api, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemGet(api, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemGet(api, "00000000-0000-0000-0000-000000000001"), 404);
  });
});

describe("itemList", () => {
  it("returns a list of items", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const items = await itemList(api);

    expect(Array.isArray(items)).toBe(true);
  });

  it("respects limit and offset", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const items = await itemList(api, 10, 0);

    expect(Array.isArray(items)).toBe(true);
    expect(items.length).toBeLessThanOrEqual(10);
  });
});

describe("itemUpdate", () => {
  it("updates an existing item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const created = await itemCreate(api, "item to update");

    const updated = await itemUpdate(api, created.id, "updated name", "updated description");
    expect(updated.id).toBe(created.id);
    expect(updated.name).toBe("updated name");
    expect(updated.description).toBe("updated description");

    await itemDelete(api, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemUpdate(api, "not-a-uuid", "name"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemUpdate(api, "00000000-0000-0000-0000-000000000000", "name"), 404);
  });

  it("returns 400 for empty name", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const created = await itemCreate(api, "item for empty name test");

    await expectStatus(itemUpdate(api, created.id, ""), 400);

    await itemDelete(api, created.id);
  });
});

describe("itemDelete", () => {
  it("deletes an existing item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    const created = await itemCreate(api, "item to delete");

    const deleted = await itemDelete(api, created.id);
    expect(deleted.id).toBe(created.id);

    await expectStatus(itemGet(api, created.id), 404);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemDelete(api, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new TemplateApi(process.env.REST_URL!);
    await expectStatus(itemDelete(api, "00000000-0000-0000-0000-000000000001"), 404);
  });
});
