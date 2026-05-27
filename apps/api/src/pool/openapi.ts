export function getPoolOpenAPISpec(serverUrl?: string) {
  return {
    openapi: "3.0.3",
    info: {
      title: "Nook API",
      description: "Nook alert aggregation and Pool event bus API",
      version: "1.0.0",
    },
    servers: serverUrl ? [{ url: serverUrl }] : [],
    tags: [
      { name: "Pool", description: "Event bus registration, WebSocket, and stats" },
      { name: "Dashboard", description: "Dashboard configuration and management" },
      { name: "Events", description: "Event log and streaming" },
      { name: "System", description: "Health and system endpoints" },
    ],
    paths: {
      "/api/pool/register": {
        post: {
          summary: "Register an app with the Pool",
          description:
            "Registers an application with the Nook Pool event bus. Returns a token for WebSocket authentication.",
          tags: ["Pool"],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["app", "secret"],
                  properties: {
                    app: { type: "string", description: "Application identifier" },
                    secret: { type: "string", description: "Shared Pool secret" },
                    events: {
                      type: "object",
                      properties: {
                        emit: {
                          type: "array",
                          items: { type: "string" },
                          description: "Event channels this app emits",
                        },
                        listen: {
                          type: "array",
                          items: { type: "string" },
                          description: "Event channels this app listens to",
                        },
                      },
                    },
                  },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "Registration successful",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      token: { type: "string", description: "WebSocket authentication token" },
                      app_id: { type: "string", description: "Assigned application ID" },
                    },
                  },
                },
              },
            },
            "400": { description: "Invalid request — app is required" },
            "403": { description: "Invalid secret" },
          },
        },
      },
      "/api/pool/ws": {
        get: {
          summary: "Pool WebSocket endpoint",
          description:
            "WebSocket connection for real-time event streaming. Requires a valid token from /api/pool/register as query parameter.",
          tags: ["Pool"],
          parameters: [
            {
              name: "token",
              in: "query",
              required: true,
              schema: { type: "string" },
              description: "Authentication token from registration",
            },
          ],
          responses: {
            "101": { description: "WebSocket upgrade successful" },
            "401": { description: "Token required" },
            "403": { description: "Invalid token" },
          },
        },
      },
      "/api/pool/stats": {
        get: {
          summary: "Pool statistics",
          description:
            "Returns current Pool connection statistics including epoch, connected app count, and app details.",
          tags: ["Pool"],
          security: [{ cookieAuth: [] }],
          responses: {
            "200": {
              description: "Pool statistics",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      epoch: { type: "string" },
                      connections: { type: "integer" },
                      apps: {
                        type: "array",
                        items: {
                          type: "object",
                          properties: {
                            app_id: { type: "string" },
                            app: { type: "string" },
                            channels: { type: "array", items: { type: "string" } },
                          },
                        },
                      },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/settings": {
        get: {
          summary: "Get settings",
          description: "Returns current Nook settings with sensitive fields redacted, runtime status, and Matrix configuration state.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          responses: {
            "200": {
              description: "Current settings",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      settings: { $ref: "#/components/schemas/AppSettings" },
                      runtime: { $ref: "#/components/schemas/RuntimeStatus" },
                      matrixConfigured: { type: "boolean" },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
        put: {
          summary: "Update settings",
          description:
            "Updates Nook settings. Redacted secret fields (••••••••) are preserved from the current config.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/AppSettings" },
              },
            },
          },
          responses: {
            "200": {
              description: "Settings updated",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      settings: { $ref: "#/components/schemas/AppSettings" },
                      runtime: { $ref: "#/components/schemas/RuntimeStatus" },
                      matrixConfigured: { type: "boolean" },
                    },
                  },
                },
              },
            },
            "400": { description: "Invalid settings or JSON body" },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/test-alert": {
        post: {
          summary: "Send test alert",
          description: "Sends a test alert through all configured delivery targets.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          responses: {
            "200": {
              description: "Test alert sent",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
            "400": { description: "Failed to send alert" },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/test-delivery-target": {
        post: {
          summary: "Test a specific delivery target",
          description: "Sends a test alert to a single delivery target by ID.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["targetId"],
                  properties: {
                    targetId: { type: "string", description: "Delivery target ID" },
                  },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "Test sent",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
            "400": { description: "targetId missing or send failed" },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/test-provider-connection": {
        post: {
          summary: "Test a provider connection",
          description: "Tests connectivity to a specific provider by ID.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["providerId"],
                  properties: {
                    providerId: { type: "string", description: "Provider ID" },
                  },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "Connection successful",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
            "400": { description: "providerId missing or connection failed" },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/events": {
        get: {
          summary: "List events",
          description: "Returns paginated event log entries with optional search and source filtering.",
          tags: ["Events"],
          security: [{ cookieAuth: [] }],
          parameters: [
            { name: "limit", in: "query", schema: { type: "integer", default: 50 }, description: "Max results" },
            { name: "offset", in: "query", schema: { type: "integer", default: 0 }, description: "Pagination offset" },
            { name: "q", in: "query", schema: { type: "string" }, description: "Search query" },
            { name: "source", in: "query", schema: { type: "string" }, description: "Filter by source" },
          ],
          responses: {
            "200": {
              description: "Event list",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      events: { type: "array", items: { type: "object" } },
                      total: { type: "integer" },
                      stats: { type: "object" },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/events/stream": {
        get: {
          summary: "Event stream (SSE)",
          description: "Server-Sent Events stream that pushes new events in real time.",
          tags: ["Events"],
          security: [{ cookieAuth: [] }],
          responses: {
            "200": {
              description: "SSE stream",
              content: {
                "text/event-stream": {
                  schema: { type: "string" },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/events/provider/{providerId}": {
        get: {
          summary: "Events by provider",
          description: "Returns paginated events filtered by provider ID.",
          tags: ["Events"],
          security: [{ cookieAuth: [] }],
          parameters: [
            { name: "providerId", in: "path", required: true, schema: { type: "string" } },
            { name: "limit", in: "query", schema: { type: "integer", default: 50 } },
            { name: "offset", in: "query", schema: { type: "integer", default: 0 } },
          ],
          responses: {
            "200": {
              description: "Provider events",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      events: { type: "array", items: { type: "object" } },
                      total: { type: "integer" },
                      stats: { type: "object" },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/events/target/{targetId}": {
        get: {
          summary: "Events by target",
          description: "Returns paginated events filtered by delivery target ID.",
          tags: ["Events"],
          security: [{ cookieAuth: [] }],
          parameters: [
            { name: "targetId", in: "path", required: true, schema: { type: "string" } },
            { name: "limit", in: "query", schema: { type: "integer", default: 50 } },
            { name: "offset", in: "query", schema: { type: "integer", default: 0 } },
          ],
          responses: {
            "200": {
              description: "Target events",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      events: { type: "array", items: { type: "object" } },
                      total: { type: "integer" },
                      stats: { type: "object" },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/events/{eventId}/replay": {
        post: {
          summary: "Replay event to target",
          description: "Re-delivers an event to a specific delivery target.",
          tags: ["Events"],
          security: [{ cookieAuth: [] }],
          parameters: [
            { name: "eventId", in: "path", required: true, schema: { type: "string" } },
          ],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["targetId"],
                  properties: {
                    targetId: { type: "string", description: "Delivery target ID" },
                  },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "Replay successful",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
            "400": { description: "targetId missing or replay failed" },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/session": {
        get: {
          summary: "Session info",
          description: "Returns current authentication status and session metadata.",
          tags: ["Dashboard"],
          security: [{ cookieAuth: [] }],
          responses: {
            "200": {
              description: "Session info",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: {
                      authenticated: { type: "boolean" },
                      passwordRequired: { type: "boolean" },
                      username: { type: "string" },
                    },
                  },
                },
              },
            },
            "401": { description: "Unauthorized" },
          },
        },
      },
      "/api/login": {
        post: {
          summary: "Login",
          description: "Authenticates with the dashboard password. Sets a session cookie on success.",
          tags: ["Dashboard"],
          requestBody: {
            required: true,
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  required: ["password"],
                  properties: {
                    password: { type: "string" },
                  },
                },
              },
            },
          },
          responses: {
            "200": {
              description: "Login successful",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
            "401": { description: "Invalid password" },
            "429": { description: "Too many attempts" },
          },
        },
      },
      "/api/logout": {
        post: {
          summary: "Logout",
          description: "Clears the session cookie.",
          tags: ["Dashboard"],
          responses: {
            "200": {
              description: "Logged out",
              content: {
                "application/json": {
                  schema: {
                    type: "object",
                    properties: { ok: { type: "boolean" } },
                  },
                },
              },
            },
          },
        },
      },
      "/health": {
        get: {
          summary: "Health check",
          tags: ["System"],
          responses: {
            "200": { description: "OK" },
          },
        },
      },
    },
    components: {
      securitySchemes: {
        cookieAuth: {
          type: "apiKey",
          in: "cookie",
          name: "nook_session",
          description: "Dashboard session cookie",
        },
      },
      schemas: {
        RuntimeStatus: {
          type: "object",
          properties: {
            serverPort: { type: "integer" },
            settingsPath: { type: "string" },
            usingPersistedSettings: { type: "boolean" },
            dashboardBuilt: { type: "boolean" },
            lastReloadedAt: { type: "string", format: "date-time" },
          },
        },
        AppSettings: {
          type: "object",
          properties: {
            delivery: {
              type: "object",
              properties: {
                items: {
                  type: "array",
                  items: { $ref: "#/components/schemas/DeliveryTarget" },
                },
              },
            },
            system: {
              type: "object",
              properties: {
                events: {
                  type: "object",
                  properties: {
                    startup: { type: "boolean" },
                    providerEnabled: { type: "boolean" },
                    providerDisabled: { type: "boolean" },
                    providerAdded: { type: "boolean" },
                    providerRemoved: { type: "boolean" },
                    deliveryFailure: { type: "boolean" },
                  },
                },
              },
            },
            timeout: { type: "integer" },
            providers: {
              type: "object",
              properties: {
                port: { type: "integer" },
                items: { type: "array", items: { type: "object" } },
              },
            },
            pool: {
              type: "object",
              properties: {
                secret: { type: "string" },
              },
            },
          },
        },
        DeliveryTarget: {
          type: "object",
          properties: {
            id: { type: "string" },
            type: {
              type: "string",
              enum: ["matrix", "discord", "webhook", "smtp", "ntfy", "perception"],
            },
            name: { type: "string" },
            enabled: { type: "boolean" },
            providerIds: { type: "array", items: { type: "string" } },
            providerTags: { type: "array", items: { type: "string" } },
          },
        },
      },
    },
  };
}

export const SCALAR_HTML = `<!doctype html>
<html>
<head>
  <title>Nook API</title>
  <meta charset="utf-8" />
</head>
<body>
  <script id="api-reference" data-url="/api/openapi"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`;
