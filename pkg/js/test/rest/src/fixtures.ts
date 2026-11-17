import { superAdminAccessToken } from "./token";

import { NarrativeEngineApi } from "@a-novel/service-narrative-engine-rest";

/** The client of the service under test, which the test runner serves at `REST_URL`. */
export const api = new NarrativeEngineApi(process.env.REST_URL!);

/** The integration super-admin's access token, which owns the projects the tests create. */
export const accessToken = await superAdminAccessToken();

/** An ID no record holds, for the requests that answer 404. */
export const ABSENT_ID = "01920000-0000-7000-8000-000000000000";

/** A beat color the service accepts. */
export const COLOR = "#4a90d9";

/** Reads a stream to its end. */
export async function collect<T>(stream: AsyncIterable<T>): Promise<T[]> {
  const items: T[] = [];
  for await (const item of stream) items.push(item);
  return items;
}
