import { AuthenticationApi, tokenCreate, tokenCreateAnon } from "@a-novel/service-authentication-rest";

/** Opens an anonymous integration session and returns its access token. */
export async function anonymousAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreateAnon(api);
  return accessToken;
}

/** Logs in as the integration super-admin and returns its access token. */
export async function superAdminAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreate(api, {
    email: process.env.SUPER_ADMIN_EMAIL ?? "noreply@agorastoryverse.com",
    password: process.env.SUPER_ADMIN_PASSWORD ?? "admin",
  });
  return accessToken;
}
