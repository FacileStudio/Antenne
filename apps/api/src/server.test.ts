import { afterEach, describe, expect, test } from "bun:test";

import { createAppServer } from "./server";
import type { AppServerOptions } from "./server-api";
import type { PoolServer } from "./pool/server";
import type { RuntimeStatus } from "./types";

const originalConsoleError = console.error;

const runtimeStatus: RuntimeStatus = {
  serverPort: 9090,
  settingsPath: "/tmp/nook-test.json",
  usingPersistedSettings: false,
  dashboardBuilt: false,
  lastReloadedAt: new Date(0).toISOString(),
};

function unusedOption(name: string) {
  return () => {
    throw new Error(`${name} should not be called in this test`);
  };
}

function createOptions(overrides: Partial<AppServerOptions> = {}): AppServerOptions {
  return {
    getSettings: unusedOption("getSettings"),
    updateSettings: unusedOption("updateSettings") as AppServerOptions["updateSettings"],
    sendConfiguredAlert: unusedOption("sendConfiguredAlert") as AppServerOptions["sendConfiguredAlert"],
    sendTestAlertToTarget: unusedOption("sendTestAlertToTarget") as AppServerOptions["sendTestAlertToTarget"],
    testProviderConnection: unusedOption("testProviderConnection") as AppServerOptions["testProviderConnection"],
    replayEventToTarget: unusedOption("replayEventToTarget") as AppServerOptions["replayEventToTarget"],
    getRuntimeStatus: () => ({ ...runtimeStatus }),
    session: { enabled: true, username: "admin", password: null, cookieName: "nook_session" },
    ...overrides,
  };
}

function captureErrors() {
  const logged: string[] = [];
  console.error = (...args: unknown[]) => {
    logged.push(args.map(String).join(" "));
  };
  return logged;
}

afterEach(() => {
  console.error = originalConsoleError;
});

describe("createAppServer error boundary", () => {
  test("turns a synchronous throw into a logged 500 instead of propagating", async () => {
    const logged = captureErrors();
    const handler = createAppServer(createOptions({
      getRuntimeStatus: () => {
        throw new Error("runtime status exploded");
      },
    }));

    const response = await handler(new Request("http://localhost/webhook/git", { method: "POST" }), null);

    expect(response.status).toBe(500);
    expect(await response.json()).toEqual({ error: { code: "internal", message: "internal server error" } });
    expect(logged.some((line) => line.includes("[server] unhandled error POST /webhook/git") && line.includes("runtime status exploded"))).toBe(true);
  });

  test("turns a rejected promise into a logged 500 instead of propagating", async () => {
    const logged = captureErrors();
    const poolServer = {
      handleRegister: async () => {
        throw new Error("pool register exploded");
      },
    } as unknown as PoolServer;
    const handler = createAppServer({ ...createOptions(), poolServer });

    const response = await handler(new Request("http://localhost/api/pool/register", { method: "POST" }), null);

    expect(response.status).toBe(500);
    expect(await response.json()).toEqual({ error: { code: "internal", message: "internal server error" } });
    expect(logged.some((line) => line.includes("[server] unhandled error POST /api/pool/register") && line.includes("pool register exploded"))).toBe(true);
  });

  test("still serves healthy routes normally", async () => {
    const handler = createAppServer(createOptions());

    const response = await handler(new Request("http://localhost/health"), null);

    expect(response.status).toBe(200);
    expect(response.headers.get("content-type")).toBe("application/json");
    expect(await response.json()).toEqual({ status: "ok" });
  });
});

describe("error envelope", () => {
  test("every error response carries {error:{code,message}}", async () => {
    const handler = createAppServer(createOptions({
      session: { enabled: true, username: "admin", password: "secret", cookieName: "nook_session" },
    }));

    const cases = [
      { request: new Request("http://localhost/api/settings"), status: 401, code: "unauthenticated" },
      { request: new Request("http://localhost/nope"), status: 404, code: "not_found" },
      { request: new Request("http://localhost/api/login", { method: "POST" }), status: 400, code: "invalid_argument" },
    ];

    for (const { request, status, code } of cases) {
      const response = await handler(request, null);
      expect(response.status).toBe(status);
      const body = await response.json() as { error?: { code?: string; message?: string } };
      expect(body.error?.code).toBe(code);
      expect(typeof body.error?.message).toBe("string");
    }
  });
});
