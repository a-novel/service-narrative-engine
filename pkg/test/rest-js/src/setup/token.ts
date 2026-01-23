import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { preRegisterUser, registerUser } from "@a-novel/service-authentication-rest-test";

export async function createUser() {
  const api = new AuthenticationApi(process.env.AUTH_API_URL!);
  const preRegister = await preRegisterUser(api, process.env.MAIL_TEST_HOST!);
  return await registerUser(api, preRegister);
}
