import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { registerSuperAdmin } from "@a-novel/service-authentication-rest-test";

/** Completes the super-admin invitation the authentication service mails at startup. */
export default async function setup(): Promise<void> {
  await registerSuperAdmin(new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!));
}
