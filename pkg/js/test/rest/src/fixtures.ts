import { AuthenticationApi } from "@a-novel/service-authentication-rest";
import { superAdminAccessToken } from "@a-novel/service-authentication-rest-test";
import { NarrativeEngineApi } from "@a-novel/service-narrative-engine-rest";

/** The client of the service under test, which the test runner serves at `REST_URL`. */
export const api = new NarrativeEngineApi(process.env.REST_URL!);

/** The client of the authentication service, which signs the suite's callers in. */
export const authApi = new AuthenticationApi(process.env.SERVICE_AUTHENTICATION_URL!);

/** The integration super-admin's access token, which owns the projects the tests create. */
export const accessToken = await superAdminAccessToken(authApi);

/** An ID no record holds, for the requests that answer 404. */
export const ABSENT_ID = "01920000-0000-7000-8000-000000000000";
