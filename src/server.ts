import { createHash, createHmac, timingSafeEqual } from "node:crypto";

import { hasConfiguredMatrixDelivery } from "./delivery";
import { getEvents, getGlobalStats, getProviderStats, getTargetStats } from "./event-log";
import type { AppSettings, DashboardSessionConfig, RuntimeStatus } from "./types";
import { handleWebhookRequest } from "./webhooks";

const dashboardDistDir = new URL("../dashboard/.output/public/", import.meta.url);
const dashboardSessionMaxAgeSeconds = 60 * 60 * 24 * 30;

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
    `Max-Age=${dashboardSessionMaxAgeSeconds}`,
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
        font-family: "Satoshi", "Helvetica Neue", Helvetica, Arial, sans-serif;
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
  const password = process.env.NOOK_ADMIN_PASSWORD ?? null;

  return {
    enabled: true,
    username: process.env.NOOK_ADMIN_USERNAME ?? "admin",
    password,
    cookieName: "nook_session",
    token: password ? createSessionToken(password) : "",
  };
}

function createSessionToken(password: string) {
  const sessionSecret = process.env.NOOK_SESSION_SECRET;

  if (sessionSecret) {
    return createHmac("sha256", sessionSecret).update(password).digest("hex");
  }

  return createHash("sha256").update(password).digest("hex");
}

export function createAppServer(options: {
  getSettings: () => AppSettings;
  updateSettings: (input: unknown) => Promise<AppSettings>;
  sendConfiguredAlert: (message: string, context?: { providerId: string; providerName: string; providerType: string; event: string }) => Promise<void>;
  sendTestAlertToTarget: (targetId: string) => Promise<void>;
  testProviderConnection: (providerId: string) => Promise<void>;
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
            matrixConfigured: hasConfiguredMatrixDelivery(options.getSettings().delivery),
          });
      }

      if (url.pathname === "/api/settings" && req.method === "PUT") {
        try {
          const body = await parseJson(req);
          const settings = await options.updateSettings(body);
          return createJsonResponse({
            settings,
            runtime: options.getRuntimeStatus(),
            matrixConfigured: hasConfiguredMatrixDelivery(settings.delivery),
          });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to save" }, { status: 400 });
        }
      }

      if (url.pathname === "/api/test-alert" && req.method === "POST") {
        const settings = options.getSettings();
        const label = settings.providers.items[0]?.name ?? "dashboard";

        try {
          await options.sendConfiguredAlert(`🧪 <b>Nook test alert</b><br/>Dashboard config for <b>${escapeHtml(label)}</b> is live.`);
          return createJsonResponse({ ok: true });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to send alert" }, { status: 400 });
        }
      }

      if (url.pathname === "/api/test-delivery-target" && req.method === "POST") {
        try {
          const body = await parseJson(req) as { targetId?: string };
          if (!body.targetId) {
            return createJsonResponse({ error: "targetId is required" }, { status: 400 });
          }

          await options.sendTestAlertToTarget(body.targetId);
          return createJsonResponse({ ok: true });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to test target" }, { status: 400 });
        }
      }

      if (url.pathname === "/api/test-provider-connection" && req.method === "POST") {
        try {
          const body = await parseJson(req) as { providerId?: string };
          if (!body.providerId) {
            return createJsonResponse({ error: "providerId is required" }, { status: 400 });
          }

          await options.testProviderConnection(body.providerId);
          return createJsonResponse({ ok: true });
        } catch (error) {
          return createJsonResponse({ error: error instanceof Error ? error.message : "failed to test provider" }, { status: 400 });
        }
      }

      if (url.pathname === "/api/events" && req.method === "GET") {
        const limit = Number(url.searchParams.get("limit")) || 50;
        return createJsonResponse({ events: getEvents({ limit }), stats: getGlobalStats() });
      }

      if (url.pathname.startsWith("/api/events/provider/") && req.method === "GET") {
        const providerId = url.pathname.replace("/api/events/provider/", "");
        const limit = Number(url.searchParams.get("limit")) || 50;
        return createJsonResponse({ events: getEvents({ providerId, limit }), stats: getProviderStats(providerId) });
      }

      if (url.pathname.startsWith("/api/events/target/") && req.method === "GET") {
        const targetId = url.pathname.replace("/api/events/target/", "");
        const limit = Number(url.searchParams.get("limit")) || 50;
        return createJsonResponse({ events: getEvents({ targetId, limit }), stats: getTargetStats(targetId) });
      }

      return createJsonResponse({ error: "not found" }, { status: 404 });
    }

    if (req.method === "POST") {
      return handleWebhookRequest(req, options.getSettings(), (message, context) => options.sendConfiguredAlert(message, context));
    }

    if (url.pathname === "/") {
      return Response.redirect(new URL("/dashboard", req.url).toString(), 302);
    }

    if (url.pathname === "/dashboard" || url.pathname.startsWith("/dashboard/")) {
      if (session.password && !isAuthenticated(req, session) && url.pathname !== "/dashboard") {
        return Response.redirect(new URL("/dashboard", req.url).toString(), 302);
      }

      if (session.password && !isAuthenticated(req, session) && url.pathname === "/dashboard") {
        return new Response(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Nook Login</title>
    <link rel="icon" type="image/webp" href="/dashboard/nook-avatar.webp">
    <style>
      @import url("https://fonts.googleapis.com/css2?family=Poppins:wght@600;700&family=Satoshi:wght@400;500;700&display=swap");
      :root {
        color-scheme: dark;
        font-family: "Satoshi", "Helvetica Neue", Helvetica, Arial, sans-serif;
        background: #050505;
        color: #fff;
      }
      *, *::before, *::after { box-sizing: border-box; }
      body {
        margin: 0;
        min-height: 100vh;
        display: grid;
        place-items: center;
        background: #050505;
        letter-spacing: -0.02em;
        padding: 1rem;
      }
      main {
        width: min(24rem, 100%);
        display: grid;
        gap: 1.5rem;
        justify-items: stretch;
        text-align: left;
      }
      .brand {
        display: flex;
        align-items: center;
        gap: 1rem;
        justify-self: start;
      }
      .brand img {
        width: 3.5rem;
        height: 3.5rem;
        border-radius: 50%;
        object-fit: cover;
        border: 2px solid rgb(255 255 255 / 0.1);
      }
      .brand span {
        font-family: "Poppins", "Helvetica Neue", Helvetica, Arial, sans-serif;
        font-weight: 700;
        font-size: 2.4rem;
        letter-spacing: -0.04em;
        line-height: 1;
      }
      p {
        margin: 0;
        color: rgb(255 255 255 / 0.5);
        line-height: 1.55;
        font-size: 0.95rem;
      }
      form {
        width: 100%;
        display: grid;
        gap: 0.75rem;
      }
      label {
        display: block;
        text-align: left;
        color: rgb(255 255 255 / 0.58);
        font-size: 0.92rem;
      }
      input {
        display: block;
        width: 100%;
        font: inherit;
        padding: 0.9rem 1.25rem;
        border-radius: 1.4rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        background: rgb(255 255 255 / 0.03);
        color: inherit;
        outline: none;
        transition: border-color 120ms ease, background 120ms ease;
      }
      input:focus {
        border-color: rgb(255 255 255 / 0.24);
        background: rgb(255 255 255 / 0.05);
      }
      button {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        gap: 0.5rem;
        width: 100%;
        font: inherit;
        padding: 0.9rem 1rem;
        border: 1px solid rgb(255 255 255 / 0.1);
        border-radius: 999px;
        background: #fff;
        color: #050505;
        font-weight: 700;
        cursor: pointer;
        transition: background 120ms ease, border-color 120ms ease;
      }
      button:hover { background: rgb(255 255 255 / 0.86); }
      button[disabled] { opacity: 0.7; cursor: wait; }
      button svg { flex: none; }
      @keyframes spin { to { transform: rotate(360deg); } }
      .spinner { animation: spin 0.7s linear infinite; }
      #error {
        color: rgb(255 131 131);
        font-size: 0.9rem;
      }
    </style>
  </head>
  <body>
    <main>
      <div class="brand">
        <img src="/dashboard/nook-avatar.webp" alt="Nook">
        <span>Nook</span>
      </div>
      <p>Sign in as <b>${escapeHtml(session.username)}</b></p>
      <form method="dialog" id="login-form">
        <label for="password">Password</label>
        <input id="password" name="password" type="password" autocomplete="current-password" autofocus>
        <button type="submit" id="submit-button">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg>
          Log in
        </button>
      </form>
      <p id="error" hidden>Wrong password.</p>
      <script>
        document.getElementById("login-form").addEventListener("submit", async (event) => {
          event.preventDefault();
          const passwordField = document.getElementById("password");
          const submitButton = document.getElementById("submit-button");
          const error = document.getElementById("error");
          error.hidden = true;
          submitButton.disabled = true;
          submitButton.innerHTML = '<svg class="spinner" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg> Loading...';
          try {
            const res = await fetch("/api/login", {
              method: "POST",
              headers: { "content-type": "application/json" },
              body: JSON.stringify({ password: passwordField.value })
            });
            if (res.ok) {
              window.location.href = "/dashboard/";
              return;
            }
          } finally {
            submitButton.disabled = false;
            submitButton.innerHTML = '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/><polyline points="10 17 15 12 10 7"/><line x1="15" y1="12" x2="3" y2="12"/></svg> Log in';
          }
          error.hidden = false;
          passwordField.focus();
          passwordField.select();
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
