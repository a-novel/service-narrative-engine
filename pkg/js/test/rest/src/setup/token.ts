import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";

export async function createUser() {
  const api = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);
  const preRegister = await preRegisterUser(api, process.env.MAIL_HOST!);
  return await registerUser(api, preRegister);
}
