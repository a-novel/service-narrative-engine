# Upgrading to NEXT

A request that fails validation answers 422 Unprocessable Entity, where it answered 400 Bad Request. The service also reads its PostgreSQL connection from the discrete `POSTGRES_*` variables only.

## Do I need to act?

**Required** for a client that branches on 400 to catch an invalid request: it reads 422 now. A client that treats every 4xx answer alike keeps working, and the release migrates no data.

**Required** for a deployment that still sets `POSTGRES_DSN`: the service no longer reads it. A deployment that sets `POSTGRES_HOST` already works as before.

## Read 422 for an invalid request

Every route answers 422 to a request that parses but breaks a rule of the contract: a blank title, a negative position, a page over 500 items, or content off its layer's schema. The body is unchanged apart from its status and title: a field that breaks its rule still answers a problem whose `tags.invalidFields` names it, with `status` 422. The service now answers as Authentication does.

400 now only means a body or parameter that does not parse, such as malformed JSON or an ID that is not a UUID. A refusal that depends on the project's state, such as a full project, still answers 409.

```diff
-if (isHttpStatusError(error, 400)) showInvalidFields(error.tags.invalidFields);
+if (isHttpStatusError(error, 422)) showInvalidFields(error.tags.invalidFields);
```

The logo write and the generation submit already answered 422, to a refused logo and to a generation that outgrows its calls. Their 422 now also answers an invalid request. A refused logo still carries the problem's `reason` tag, and an outgrown generation now carries one too, `contextTooLarge`, so a client tells "narrow the request" from "fix the request". A generation on a project past its 16 MiB budget answers the same 422 with a `limit` tag of `projectBytes` instead: no narrower request fits until the project shrinks. An invalid request whose rule spans several fields, such as a range past the cell's end or a repeated target, answers the status text alone.

## Set the discrete PostgreSQL variables

`POSTGRES_DSN` was the fallback when `POSTGRES_HOST` was empty. Set the fields its URL carried instead:

| Variable               | From the URL                                                                        |
| ---------------------- | ----------------------------------------------------------------------------------- |
| `POSTGRES_HOST`        | The host.                                                                           |
| `POSTGRES_PORT`        | The port, when it is not `5432`.                                                    |
| `POSTGRES_USER`        | The user.                                                                           |
| `POSTGRES_PASSWORD`    | The password, as a secret.                                                          |
| `POSTGRES_DATABASE`    | The database, the URL's path.                                                       |
| `POSTGRES_TLS_ENABLED` | `false` only where the URL said `sslmode=disable`; the variable defaults to `true`. |

The previous release reads these variables too, so set them before you upgrade. The server and the migrations job start once the connection works.

## Notes

`MAX_CONCURRENT_PROJECT_READS` keeps its default: unset, the service runs 4 whole-project reads per GiB of `GOMEMLIMIT`, or one per CPU without a memory limit. An explicit `0` or a negative value now sizes the reads the same way, where it ran one read at a time before: set `1` to keep that.
