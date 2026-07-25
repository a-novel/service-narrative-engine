import { anonymousAccessToken, superAdminAccessToken } from "./token";

import { beforeAll, describe, expect, it } from "vitest";

import { expectStatus } from "@a-novel-kit/nodelib-test/http";
import {
  NarrativeEngineApi,
  itemCreate,
  itemDelete,
  itemGet,
  itemList,
  itemUpdate,
} from "@a-novel/service-narrative-engine-rest";

let accessToken: string;

beforeAll(async () => {
  accessToken = await superAdminAccessToken();
});

describe("itemCreate", () => {
  it("creates a new item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const item = await itemCreate(api, accessToken, "test item", "test description");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("test item");
    expect(item.description).toBe("test description");
    expect(item.createdAt).toBeTruthy();
    expect(item.updatedAt).toBeTruthy();

    await itemDelete(api, accessToken, item.id);
  });

  it("creates an item without description", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const item = await itemCreate(api, accessToken, "no description item");

    expect(item.id).toBeTruthy();
    expect(item.name).toBe("no description item");

    await itemDelete(api, accessToken, item.id);
  });

  it("returns 400 for empty name", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemCreate(api, accessToken, ""), 400);
  });
});

describe("itemGet", () => {
  it("retrieves an existing item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const created = await itemCreate(api, accessToken, "item to get");

    const item = await itemGet(api, accessToken, created.id);
    expect(item.id).toBe(created.id);
    expect(item.name).toBe("item to get");

    await itemDelete(api, accessToken, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemGet(api, accessToken, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemGet(api, accessToken, "00000000-0000-0000-0000-000000000001"), 404);
  });
});

describe("itemList", () => {
  it("returns a list of items", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const items = await itemList(api, accessToken);

    expect(Array.isArray(items)).toBe(true);
  });

  it("respects limit and offset", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const items = await itemList(api, accessToken, 10, 0);

    expect(Array.isArray(items)).toBe(true);
    expect(items.length).toBeLessThanOrEqual(10);
  });

  it("returns 401 with a bearer challenge when the token is missing", async () => {
    const response = await fetch(`${process.env.REST_URL!}/items`);

    expect(response.status).toBe(401);
    expect(response.headers.get("www-authenticate")).toBe(`Bearer realm="narrative-engine", error="invalid_token"`);
  });

  it("returns 403 when the token lacks the read permission", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const anonymousToken = await anonymousAccessToken();

    await expectStatus(itemList(api, anonymousToken), 403);
  });
});

describe("itemUpdate", () => {
  it("updates an existing item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const created = await itemCreate(api, accessToken, "item to update");

    const updated = await itemUpdate(api, accessToken, created.id, "updated name", "updated description");
    expect(updated.id).toBe(created.id);
    expect(updated.name).toBe("updated name");
    expect(updated.description).toBe("updated description");

    await itemDelete(api, accessToken, created.id);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemUpdate(api, accessToken, "not-a-uuid", "name"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemUpdate(api, accessToken, "00000000-0000-0000-0000-000000000000", "name"), 404);
  });

  it("returns 400 for empty name", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const created = await itemCreate(api, accessToken, "item for empty name test");

    await expectStatus(itemUpdate(api, accessToken, created.id, ""), 400);

    await itemDelete(api, accessToken, created.id);
  });
});

describe("itemDelete", () => {
  it("deletes an existing item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    const created = await itemCreate(api, accessToken, "item to delete");

    const deleted = await itemDelete(api, accessToken, created.id);
    expect(deleted.id).toBe(created.id);

    await expectStatus(itemGet(api, accessToken, created.id), 404);
  });

  it("returns 400 for invalid ID format", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemDelete(api, accessToken, "not-a-uuid"), 400);
  });

  it("returns 404 for non-existent item", async () => {
    const api = new NarrativeEngineApi(process.env.REST_URL!);
    await expectStatus(itemDelete(api, accessToken, "00000000-0000-0000-0000-000000000001"), 404);
  });
});
