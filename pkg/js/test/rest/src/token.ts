import { AuthenticationApi, tokenCreate } from "@a-novel/service-authentication-rest";

/** Logs in as the integration super-admin and returns its access token. */
export async function superAdminAccessToken(): Promise<string> {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const { accessToken } = await tokenCreate(api, {
    email: process.env.SUPER_ADMIN_EMAIL!,
    password: process.env.SUPER_ADMIN_PASSWORD!,
  });
  return accessToken;
}
