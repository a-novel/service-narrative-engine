import { setTimeout as delay } from "node:timers/promises";

import { isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import {
  AuthenticationApi,
  Role,
  claimsGet,
  credentialsCreate,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

interface MailpitSearchResponse {
  messages: Array<{ ID: string }>;
}

interface MailpitMessage {
  HTML: string;
}

/** Opens an anonymous integration session and returns its access token. */
export async function anonymousAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreateAnon(api);
  return accessToken;
}

/** Logs in as the integration super-admin, completing its initial invitation when needed. */
export async function superAdminAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const email = process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com";
  const password = process.env.SUPER_ADMIN_PASSWORD ?? "admin";

  try {
    const { accessToken } = await tokenCreate(api, { email, password });
    return accessToken;
  } catch (error) {
    if (!isHttpStatusError(error, 401)) throw error;
  }

  const registrationURL = await waitForRegistration(email);
  const shortCode = registrationURL.searchParams.get("shortCode");
  const target = registrationURL.searchParams.get("target");
  if (!shortCode || !target || Buffer.from(target, "base64url").toString() !== email) {
    throw new Error("maintenance registration link is invalid");
  }

  const anonymousToken = await tokenCreateAnon(api);
  const token = await credentialsCreate(api, anonymousToken.accessToken, { email, password, shortCode });
  const claims = await claimsGet(api, token.accessToken);
  if (claims.roles?.length !== 1 || claims.roles[0] !== Role.SuperAdmin) {
    throw new Error("maintenance registration did not assign the super-admin role");
  }

  return token.accessToken;
}

async function waitForRegistration(email: string): Promise<URL> {
  const mailURL = process.env.MAIL_UI_URL ?? `http://localhost:${requiredEnvironment("MAIL_UI_PORT")}`;
  const deadline = Date.now() + 15_000;

  while (Date.now() < deadline) {
    const query = new URLSearchParams({ query: `to:"${email}"`, limit: "10" });
    const response = await fetch(`${mailURL}/api/v1/search?${query}`);
    if (!response.ok) throw new Error(`mail search failed with ${response.status}`);

    const { messages } = (await response.json()) as MailpitSearchResponse;
    for (const { ID } of messages) {
      const messageResponse = await fetch(`${mailURL}/api/v1/message/${ID}`);
      if (!messageResponse.ok) throw new Error(`mail read failed with ${messageResponse.status}`);

      const { HTML } = (await messageResponse.json()) as MailpitMessage;
      const document = new DOMParser().parseFromString(HTML, "text/html");
      const href = Array.from(document.querySelectorAll<HTMLAnchorElement>("a[href]"))
        .map((link) => link.href)
        .find((link) => new URL(link).pathname === "/ext/account/create");
      if (href) return new URL(href);
    }

    await delay(100);
  }

  throw new Error(`maintenance registration email was not delivered to ${email}`);
}

function requiredEnvironment(name: string): string {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required for REST integration tests`);
  return value;
}
