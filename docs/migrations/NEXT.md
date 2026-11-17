# Upgrading to NEXT

A request that fails validation answers 422 Unprocessable Entity, where it answered 400 Bad Request.

## Do I need to act?

**Required** for a client that branches on 400 to catch an invalid request: it reads 422 now. A client that treats every 4xx answer alike keeps working, and the release migrates no data.

## Read 422 for an invalid request

Every route answers 422 to a request that parses but breaks a rule of the contract: a blank title, a negative position, a page over 500 items, or content off its layer's schema. The body is unchanged apart from its status: a field that breaks its rule still answers a problem whose `tags.invalidFields` names it, with `status` 422. The service now answers as Authentication does.

400 now only means a body or parameter that does not parse, such as malformed JSON or an ID that is not a UUID. A refusal that depends on the project's state, such as a full project, still answers 409.

```diff
-if (isHttpStatusError(error, 400)) showInvalidFields(error.tags.invalidFields);
+if (isHttpStatusError(error, 422)) showInvalidFields(error.tags.invalidFields);
```

The logo write and the generation submit already answered 422, to a refused logo and to a generation that outgrows its calls. Their 422 now also answers an invalid request. A refused logo still carries the problem's `reason` tag.
