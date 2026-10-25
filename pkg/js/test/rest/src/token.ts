import { AuthenticationApi, tokenCreate, tokenCreateAnon } from "@a-novel/service-authentication-rest";

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
