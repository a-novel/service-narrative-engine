import { randomUUID } from "node:crypto";
import { setTimeout as delay } from "node:timers/promises";

import { mailLinks, searchMail } from "@a-novel-kit/nodelib-test/mailpit";
import {
  AuthenticationApi,
  Lang,
  credentialsCreate,
  shortCodeCreateRegister,
  tokenCreate,
  tokenCreateAnon,
} from "@a-novel/service-authentication-rest";

/** Returns the integration super-admin's sign-in, which the suite's global setup completes once. */
export function superAdminCredentials(): { email: string; password: string } {
  return {
    email: process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com",
    password: process.env.SUPER_ADMIN_PASSWORD ?? "admin",
  };
}

/** Returns the base URL of the Mailpit API that receives the authentication service's mail. */
export function mailURL(): string {
  return process.env.MAIL_UI_URL!;
}

/** Opens an anonymous session and returns its access token. Such a session owns no project. */
export async function anonymousAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreateAnon(api);
  return accessToken;
}

/** Signs in as the integration super-admin, a user like any other to this service. */
export async function superAdminAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreate(api, superAdminCredentials());
  return accessToken;
}

/**
 * Registers a new account through the super-admin's invitation, and returns its access token. The account
 * holds the user role, whose caps limit its generations.
 */
export async function userAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const email = `${randomUUID()}@agorastoryverse.com`;
  await shortCodeCreateRegister(api, await superAdminAccessToken(), { email, lang: Lang.En });

  const shortCode = await registrationShortCode(email);
  const anonymousToken = await tokenCreateAnon(api);
  const { accessToken } = await credentialsCreate(api, anonymousToken.accessToken, {
    email,
    password: "integration-user",
    shortCode,
  });

  return accessToken;
}

/**
 * Waits up to 15 seconds for the invitation mailed to email, and returns the short code of its link. It polls
 * Mailpit itself: the suite's global setup calls it outside any test, where `vi.waitFor` cannot run.
 */
export async function registrationShortCode(email: string): Promise<string> {
  const deadline = Date.now() + 15_000;

  while (Date.now() < deadline) {
    const links = (await searchMail(mailURL(), `to:"${email}"`)).flatMap(({ HTML }) => mailLinks(HTML));
    const link = links.find((href) => new URL(href).pathname === "/ext/account/create");

    if (link) {
      const shortCode = new URL(link).searchParams.get("shortCode");
      if (!shortCode) throw new Error("the invitation link carries no short code");
      return shortCode;
    }

    await delay(100);
  }

  throw new Error(`no invitation reached ${email}`);
}
