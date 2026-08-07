import http from "node:http";

let latestRequest;

function sendJSON(response, status, body) {
  response.writeHead(status, { "Content-Type": "application/json" });
  response.end(JSON.stringify(body));
}

async function readJSON(request) {
  const chunks = [];

  for await (const chunk of request) chunks.push(chunk);

  return JSON.parse(Buffer.concat(chunks).toString("utf8"));
}

const server = http.createServer(async (request, response) => {
  if (request.method === "GET" && request.url === "/health") {
    sendJSON(response, 200, { status: "up" });

    return;
  }

  if (request.method === "GET" && request.url === "/requests/latest") {
    if (latestRequest === undefined) {
      sendJSON(response, 404, { error: "no request recorded" });

      return;
    }

    sendJSON(response, 200, latestRequest);

    return;
  }

  if (request.method === "POST" && request.url === "/v1/responses") {
    try {
      latestRequest = await readJSON(request);
    } catch {
      sendJSON(response, 400, { error: "invalid JSON" });

      return;
    }

    const proposal = {
      accepted: true,
      source: "controlled-provider",
      paragraph: "The answering light moved beneath the waves.",
    };

    sendJSON(response, 200, {
      id: "resp_narrative_integration",
      status: "completed",
      model: "gpt-5.6-terra-integration",
      output: [
        {
          id: "msg_narrative_integration",
          type: "message",
          status: "completed",
          role: "assistant",
          content: [
            {
              type: "output_text",
              text: JSON.stringify(proposal),
              annotations: [],
            },
          ],
        },
      ],
      usage: {
        input_tokens: 100,
        input_tokens_details: { cached_tokens: 0 },
        output_tokens: 20,
        output_tokens_details: { reasoning_tokens: 5 },
        total_tokens: 120,
      },
    });

    return;
  }

  sendJSON(response, 404, { error: "route not found" });
});

server.listen(8080, "0.0.0.0");

function shutdown() {
  server.close(() => process.exit(0));
}

process.on("SIGINT", shutdown);
process.on("SIGTERM", shutdown);
