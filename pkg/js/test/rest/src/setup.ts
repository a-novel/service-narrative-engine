import { registrationShortCode, superAdminAccessToken, superAdminCredentials } from "./token";

import { isHttpStatusError } from "@a-novel-kit/nodelib-browser/http";
import { AuthenticationApi, credentialsCreate, tokenCreateAnon } from "@a-novel/service-authentication-rest";

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
  const shortCode = await registrationShortCode(email);

  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const anonymousToken = await tokenCreateAnon(api);
  await credentialsCreate(api, anonymousToken.accessToken, { email, password, shortCode });
}
