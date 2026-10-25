import { describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import { TemplateApi, itemCreate, itemDelete, itemGet, itemList, itemUpdate } from "@a-novel/service-template-rest";

const api = new TemplateApi(process.env.REST_URL!);

describe("itemCreate", () => {
  it("creates a new item", async () => {
    const item = await itemCreate(api, "test item", "test description");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("test item");
    expect(item.description).toBe("test description");
    expect(item.createdAt).toBeTruthy();
    expect(item.updatedAt).toBeTruthy();

    await itemDelete(api, item.id);
  });

  it("creates an item without description", async () => {
    const item = await itemCreate(api, "no description item");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("no description item");

    await itemDelete(api, item.id);
  });

  it("returns 400 for empty name", async () => {
    await expectStatus(itemCreate(api, ""), 400);
  });

  it("names the invalid field of a rejected item", async () => {
    await expect(itemCreate(api, "   ")).rejects.toThrow(
      '{"type":"about:blank","title":"Bad Request","status":400,"tags":{"invalidFields":{"name":"notblank"}}}'
    );
  });
});

describe("itemGet", () => {
  it("retrieves an existing item", async () => {
    const created = await itemCreate(api, "item to get");

    const item = await itemGet(api, created.id);
    expect(item.id).toBe(created.id);
    expect(item.name).toBe("item to get");

    await itemDelete(api, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    await expectStatus(itemGet(api, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    await expectStatus(itemGet(api, "00000000-0000-0000-0000-000000000001"), 404);
  });
});

describe("itemList", () => {
  it("lists a created item", async () => {
    const created = await itemCreate(api, "item to list");

    const items = await itemList(api);
    expect(items.map((item) => item.id)).toContain(created.id);

    await itemDelete(api, created.id);
  });

  it("returns at most limit items", async () => {
    const created = await itemCreate(api, "item to page");

    const items = await itemList(api, 1);
    expect(items).toHaveLength(1);

    await itemDelete(api, created.id);
  });

  it("continues after the given item", async () => {
    const older = await itemCreate(api, "older item");
    const newer = await itemCreate(api, "newer item");

    const items = await itemList(api, 1, newer.id);
    expect(items.map((item) => item.id)).toEqual([older.id]);

    await itemDelete(api, newer.id);
    await itemDelete(api, older.id);
  });
});

describe("itemUpdate", () => {
  it("updates an existing item", async () => {
    const created = await itemCreate(api, "item to update");

    const updated = await itemUpdate(api, created.id, "updated name", "updated description");
    expect(updated.id).toBe(created.id);
    expect(updated.name).toBe("updated name");
    expect(updated.description).toBe("updated description");

    await itemDelete(api, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    await expectStatus(itemUpdate(api, "not-a-uuid", "name"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    await expectStatus(itemUpdate(api, "00000000-0000-0000-0000-000000000001", "name"), 404);
  });

  it("returns 400 for empty name", async () => {
    const created = await itemCreate(api, "item for empty name test");

    await expectStatus(itemUpdate(api, created.id, ""), 400);

    await itemDelete(api, created.id);
  });
});

describe("itemDelete", () => {
  it("deletes an existing item", async () => {
    const created = await itemCreate(api, "item to delete");

    const deleted = await itemDelete(api, created.id);
    expect(deleted.id).toBe(created.id);

    await expectStatus(itemGet(api, created.id), 404);
  });

  it("returns 400 for invalid ID format", async () => {
    await expectStatus(itemDelete(api, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    await expectStatus(itemDelete(api, "00000000-0000-0000-0000-000000000001"), 404);
  });
});
