import { randomBytes, timingSafeEqual } from "node:crypto";

import { hasMatrixConfig } from "./matrix";
import type { AppSettings, DashboardSessionConfig, RuntimeStatus } from "./types";
import { handleWebhookRequest } from "./webhooks";

const dashboardDistDir = new URL("../dashboard/.output/public/", import.meta.url);

function escapeHtml(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

function getCookie(req: Request, name: string) {
  const cookieHeader = req.headers.get("cookie");
  if (!cookieHeader) {
    return null;
  }

  for (const segment of cookieHeader.split(";")) {
    const [rawName, ...rawValue] = segment.trim().split("=");
    if (rawName === name) {
      return decodeURIComponent(rawValue.join("="));
    }
  }

  return null;
}

function safeEquals(left: string, right: string) {
  const leftBuffer = Buffer.from(left);
  const rightBuffer = Buffer.from(right);
  if (leftBuffer.length !== rightBuffer.length) {
    return false;
  }

  return timingSafeEqual(leftBuffer, rightBuffer);
}

function shouldUseSecureCookies(req: Request) {
  const url = new URL(req.url);
  if (url.protocol === "https:") {
    return true;
  }

  return req.headers.get("x-forwarded-proto") === "https";
}

function buildSessionCookie(req: Request, session: DashboardSessionConfig) {
  const flags = [
    `${session.cookieName}=${encodeURIComponent(session.token)}`,
    "Path=/",
    "HttpOnly",
    "SameSite=Lax",
  ];

  if (shouldUseSecureCookies(req)) {
    flags.push("Secure");
  }

  return flags.join("; ");
}

function buildExpiredSessionCookie(req: Request, session: DashboardSessionConfig) {
  const flags = [
    `${session.cookieName}=`,
    "Path=/",
    "Max-Age=0",
    "HttpOnly",
    "SameSite=Lax",
  ];

  if (shouldUseSecureCookies(req)) {
    flags.push("Secure");
  }

  return flags.join("; ");
}

function createJsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
}

function isAuthenticated(req: Request, session: DashboardSessionConfig) {
  if (!session.enabled || !session.password) {
    return true;
  }

  const cookie = getCookie(req, session.cookieName);
  if (!cookie) {
    return false;
  }

  return safeEquals(cookie, session.token);
}

function unauthorizedResponse() {
  return createJsonResponse({ error: "unauthorized" }, { status: 401 });
}

function buildFallbackDashboardHtml() {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nook Dashboard</title>
    <style>
      :root {
        color-scheme: dark;
        font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
        background: #050505;
        color: #fff;
      }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background: #050505;
        letter-spacing: -0.02em;
      }
      main {
        width: min(42rem, calc(100vw - 2rem));
        padding: 2rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        border-radius: 1.5rem;
        background: rgb(255 255 255 / 0.03);
        box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.02);
      }
      h1 {
        margin-top: 0;
        font-size: 2rem;
      }
      p {
        color: rgb(255 255 255 / 0.58);
        line-height: 1.5;
      }
      code {
        padding: 0.2rem 0.45rem;
        border-radius: 999px;
        background: rgb(255 255 255 / 0.05);
      }
    </style>
  </head>
  <body>
    <main>
      <h1>Nook dashboard not built</h1>
      <p>Run <code>bun run dashboard:build</code> before serving the dashboard. Backend, monitors, and webhooks still work.</p>
    </main>
  </body>
</html>`;
}

async function serveDashboardAsset(pathname: string, dashboardBuilt: boolean) {
  if (!dashboardBuilt) {
    return new Response(buildFallbackDashboardHtml(), {
      headers: { "Content-Type": "text/html; charset=utf-8" },
    });
  }

  const relativePath = pathname === "/dashboard" || pathname === "/dashboard/"
    ? "index.html"
    : pathname.replace(/^\/dashboard\/?/, "");
  const file = Bun.file(new URL(relativePath || "index.html", dashboardDistDir));

  if (relativePath !== "index.html" && !(await file.exists())) {
    return serveDashboardAsset("/dashboard/index.html", dashboardBuilt);
  }

  return new Response(file);
}

async function parseJson(req: Request) {
  try {
    return await req.json();
  } catch {
    throw new Error("Invalid JSON body.");
  }
}

export function createDashboardSessionConfig(): DashboardSessionConfig {
  return {
    enabled: true,
    username: process.env.NOOK_ADMIN_USERNAME ?? "admin",
    password: process.env.NOOK_ADMIN_PASSWORD ?? null,
    cookieName: "nook_session",
    token: randomBytes(32).toString("hex"),
  };
}

export function createAppServer(options: {
  getSettings: () => AppSettings;
  updateSettings: (input: unknown) => Promise<AppSettings>;
  sendConfiguredAlert: (message: string) => Promise<void>;
  getRuntimeStatus: () => RuntimeStatus;
  session: DashboardSessionConfig;
}) {
  return async (req: Request): Promise<Response> => {
    const url = new URL(req.url);
    const { session } = options;
    const dashboardBuilt = options.getRuntimeStatus().dashboardBuilt;

    if (url.pathname === "/health") {
      return new Response("ok", { status: 200 });
    }

    if (url.pathname === "/api/login" && req.method === "POST") {
      if (!session.password) {
        return createJsonResponse({ ok: true });
      }

      let body: { password?: string };
      try {
        body = await parseJson(req) as { password?: string };
      } catch (error) {
        return createJsonResponse({ error: error instanceof Error ? error.message : "bad request" }, { status: 400 });
      }

      if (!body.password || !safeEquals(body.password, session.password)) {
        return createJsonResponse({ error: "invalid password" }, { status: 401 });
      }

      return createJsonResponse(
        { ok: true },
        {
          headers: {
            "Set-Cookie": buildSessionCookie(req, session),
          },
        },
      );
    }

    if (url.pathname === "/api/logout" && req.method === "POST") {
      return createJsonResponse(
        { ok: true },
        {
          headers: {
            "Set-Cookie": buildExpiredSessionCookie(req, session),
          },
        },
      );
    }

    if (url.pathname.startsWith("/api/")) {
      if (!isAuthenticated(req, session)) {
        return unauthorizedResponse();
      }

      if (url.pathname === "/api/session" && req.method === "GET") {
        return createJsonResponse({
          authenticated: isAuthenticated(req, session),
          passwordRequired: Boolean(session.password),
          username: session.username,
        });
      }

      if (url.pathname === "/api/settings" && req.method === "GET") {
        return createJsonResponse({
          settings: options.getSettings(),
          runtime: options.getRuntimeStatus(),
          matrixConfigured: hasMatrixConfig(options.getSettings().matrix),
        });
      }

      if (url.pathname === "/api/settings" && req.method === "PUT") {
        try {
          const body = await parseJson(req);
          const settings = await options.updateSettings(body);
          return createJsonResponse({
            settings,
            runtime: options.getRuntimeStatus(),
            matrixConfigured: hasMatrixConfig(settings.matrix),
          });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to save" }, { status: 400 });
        }
      }

      if (url.pathname === "/api/test-alert" && req.method === "POST") {
        const settings = options.getSettings();
        const label = settings.sites[0]?.name ?? "dashboard";

        try {
          await options.sendConfiguredAlert(`🧪 <b>Nook test alert</b><br/>Dashboard config for <b>${escapeHtml(label)}</b> is live.`);
          return createJsonResponse({ ok: true });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to send alert" }, { status: 400 });
        }
      }

      return createJsonResponse({ error: "not found" }, { status: 404 });
    }

    if (url.pathname.startsWith("/webhook/")) {
      return handleWebhookRequest(req, options.getSettings(), options.sendConfiguredAlert);
    }

    if (url.pathname === "/") {
      return Response.redirect(new URL("/dashboard", req.url).toString(), 302);
    }

    if (url.pathname === "/dashboard" || url.pathname.startsWith("/dashboard/")) {
      if (session.password && !isAuthenticated(req, session) && url.pathname === "/dashboard") {
        return new Response(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nook Login</title>
    <style>
      :root {
        color-scheme: dark;
        font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
        background: #050505;
        color: #fff;
      }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background: #050505;
        letter-spacing: -0.02em;
      }
      main {
        width: min(26rem, calc(100vw - 2rem));
        padding: 2rem;
        border-radius: 1.4rem;
        background: rgb(255 255 255 / 0.03);
        border: 1px solid rgb(255 255 255 / 0.1);
        box-shadow: inset 0 0 0 1px rgb(255 255 255 / 0.02);
      }
      h1 { margin-top: 0; }
      label, input, button { display: block; width: 100%; }
      input {
        margin: 1rem 0;
        padding: 0.9rem 1.25rem;
        border-radius: 1.4rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        background: rgb(255 255 255 / 0.03);
        color: inherit;
        outline: none;
      }
      button {
        padding: 0.9rem 1rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        border-radius: 999px;
        background: #fff;
        color: #050505;
        font-weight: 700;
      }
      p { color: rgb(255 255 255 / 0.58); }
    </style>
  </head>
  <body>
    <main>
      <h1>Nook dashboard</h1>
      <p>Password gate enabled for <b>${escapeHtml(session.username)}</b>.</p>
      <form method="dialog" id="login-form">
        <label for="password">Password</label>
        <input id="password" name="password" type="password" autocomplete="current-password">
        <button type="submit">Enter dashboard</button>
      </form>
      <p id="error" hidden>Wrong password.</p>
      <script>
        document.getElementById("login-form").addEventListener("submit", async (event) => {
          event.preventDefault();
          const password = document.getElementById("password").value;
          const res = await fetch("/api/login", {
            method: "POST",
            headers: { "content-type": "application/json" },
            body: JSON.stringify({ password })
          });
          if (res.ok) {
            window.location.reload();
            return;
          }
          document.getElementById("error").hidden = false;
        });
      </script>
    </main>
  </body>
</html>`, {
          headers: { "Content-Type": "text/html; charset=utf-8" },
        });
      }

      return serveDashboardAsset(url.pathname, dashboardBuilt);
    }

    return new Response("not found", { status: 404 });
  };
}
