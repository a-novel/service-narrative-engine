import { mailURL, superAdminAccessToken, superAdminCredentials } from "./token";

import { setTimeout as delay } from "node:timers/promises";

import { isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import { AuthenticationApi, credentialsCreate, tokenCreateAnon } from "@a-novel/service-authentication-rest";

/** A Mailpit search result, reduced to the message IDs. */
interface MailpitSearchResponse {
  messages: Array<{ ID: string }>;
}

/** A Mailpit message, reduced to its HTML body. */
interface MailpitMessage {
  HTML: string;
}

/** Matches the account-creation link of an invitation email, its `&` still escaped as in HTML. */
const REGISTRATION_LINK = /href="([^"]*\/ext\/account\/create\?[^"]*)"/;

/**
 * Completes the invitation the authentication service mails the integration super-admin at startup.
 * It runs once before every test file, which then signs in with the super-admin's credentials.
 */
export default async function setup(): Promise<void> {
  try {
    await superAdminAccessToken();

    return;
  } catch (error) {
    // A fresh integration database holds an invitation instead of credentials.
    if (!isHttpStatusError(error, 401)) throw error;
  }

  const { email, password } = superAdminCredentials();
  const shortCode = (await waitForRegistration(email)).searchParams.get("shortCode");
  if (!shortCode) throw new Error("the invitation link carries no short code");

  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const anonymousToken = await tokenCreateAnon(api);
  await credentialsCreate(api, anonymousToken.accessToken, { email, password, shortCode });
}

/** Waits up to 15 seconds for the invitation mailed to email, and returns its account-creation link. */
async function waitForRegistration(email: string): Promise<URL> {
  const deadline = Date.now() + 15_000;

  while (Date.now() < deadline) {
    const query = new URLSearchParams({ query: `to:"${email}"`, limit: "10" });
    const { messages } = (await fetch(`${mailURL()}/api/v1/search?${query}`).then(readJSON)) as MailpitSearchResponse;

    for (const { ID } of messages) {
      const { HTML } = (await fetch(`${mailURL()}/api/v1/message/${ID}`).then(readJSON)) as MailpitMessage;
      const link = REGISTRATION_LINK.exec(HTML)?.[1];
      if (link) return new URL(link.replaceAll("&amp;", "&"));
    }

    await delay(100);
  }

  throw new Error(`no invitation reached ${email}`);
}

/** Returns the JSON body of a successful Mailpit response. */
async function readJSON(response: Response): Promise<unknown> {
  if (!response.ok) throw new Error(`Mailpit answered ${response.status}`);
  return await response.json();
}
