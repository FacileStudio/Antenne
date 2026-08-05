import {
  buildExpiredSessionCookie,
  buildSessionCookie,
  clearLoginFailures,
  createDashboardSessionConfig,
  isAuthenticated,
  isLoginRateLimited,
  isValidSessionPassword,
  recordLoginFailure,
} from "./server-auth";
import { handleDashboardApiRequest, type AppServerOptions } from "./server-api";
import { isDashboardAssetPath, serveDashboardAsset } from "./server-dashboard";
import { renderLoginPage } from "./server-login-page";
import { createErrorResponse, createJsonResponse } from "./server-responses";
import { handleWebhookRequest } from "./webhooks";
import type { PoolServer } from "./pool/server";
import { getPoolOpenAPISpec, SCALAR_HTML } from "./pool/openapi";

function getClientIp(req: Request) {
  return req.headers.get("x-forwarded-for")?.split(",")[0]?.trim()
    ?? req.headers.get("cf-connecting-ip")
    ?? req.headers.get("x-real-ip")
    ?? "unknown";
}

function unauthorizedResponse() {
  return createErrorResponse("unauthenticated", "unauthorized");
}

function redirectTo(path: string, req: Request, init?: ResponseInit) {
  return new Response(null, {
    ...init,
    status: init?.status ?? 302,
    headers: {
      Location: new URL(path, req.url).toString(),
      "Cache-Control": "private, no-store",
      Vary: "Cookie",
      ...(init?.headers ?? {}),
    },
  });
}

async function handleLoginForm(req: Request, options: AppServerOptions) {
  if (!options.session.password) {
    return redirectTo("/dashboard/", req, { status: 303 });
  }

  if (isLoginRateLimited(req)) {
    console.warn(`[auth] login rate-limited ip=${getClientIp(req)}`);
    return redirectTo("/dashboard?error=rate-limit", req, { status: 303 });
  }

  const formData = await req.formData();
  const password = formData.get("password");

  if (!isValidSessionPassword(typeof password === "string" ? password : null, options.session)) {
    recordLoginFailure(req);
    console.warn(`[auth] login failed ip=${getClientIp(req)}`);
    return redirectTo("/dashboard?error=invalid", req, { status: 303 });
  }

  clearLoginFailures(req);
  console.log(`[auth] login success ip=${getClientIp(req)}`);

  return redirectTo("/dashboard/providers", req, {
    status: 303,
    headers: {
      "Set-Cookie": buildSessionCookie(req, options.session),
    },
  });
}

function renderLoginResponse(options: AppServerOptions, errorMessage?: string) {
  return new Response(renderLoginPage(options.session, errorMessage), {
    headers: {
      "Content-Type": "text/html; charset=utf-8",
      "Cache-Control": "private, no-store",
      Vary: "Cookie",
    },
  });
}

export { createDashboardSessionConfig } from "./server-auth";

function createRequestHandler(options: AppServerOptions & { poolServer?: PoolServer }) {
  return async (req: Request, server: unknown): Promise<Response> => {
    const url = new URL(req.url);
    const dashboardBuilt = options.getRuntimeStatus().dashboardBuilt;

    if (options.poolServer) {
      if (url.pathname === "/api/pool/register" && req.method === "POST") {
        return options.poolServer.handleRegister(req);
      }

      if (url.pathname === "/api/pool/ws" && req.headers.get("upgrade")?.toLowerCase() === "websocket") {
        const result = options.poolServer.handleUpgrade(req, server as any);
        if (!result) return new Response(null, { status: 101 });
        return result;
      }

      if (url.pathname === "/api/pool/stats" && req.method === "GET") {
        if (!isAuthenticated(req, options.session)) {
          return unauthorizedResponse();
        }
        return createJsonResponse(options.poolServer.getStats());
      }
    }

    if (url.pathname === "/health") {
      return createJsonResponse({ status: "ok" });
    }

    if (url.pathname === "/api/docs" && req.method === "GET") {
      return new Response(SCALAR_HTML, {
        headers: { "Content-Type": "text/html; charset=utf-8" },
      });
    }

    if (url.pathname === "/api/openapi" && req.method === "GET") {
      const spec = getPoolOpenAPISpec(`${url.protocol}//${url.host}`);
      return Response.json(spec);
    }

    if (url.pathname === "/favicon.ico") {
      return serveDashboardAsset("/dashboard/favicon.ico", dashboardBuilt);
    }

    if (url.pathname === "/api/logout" && req.method === "POST") {
      return createJsonResponse(
        { ok: true },
        {
          headers: {
            "Set-Cookie": buildExpiredSessionCookie(req, options.session),
          },
        },
      );
    }

    if (url.pathname === "/api/login" && req.method === "POST") {
      if (!options.session.password) {
        return createJsonResponse({ ok: true });
      }

      if (isLoginRateLimited(req)) {
        console.warn(`[auth] api login rate-limited ip=${getClientIp(req)}`);
        return createErrorResponse("rate_limited", "too many attempts");
      }

      let body: { password?: string };
      try {
        body = await req.json() as { password?: string };
      } catch {
        return createErrorResponse("invalid_argument", "Invalid JSON body.");
      }

      if (!isValidSessionPassword(body.password, options.session)) {
        recordLoginFailure(req);
        console.warn(`[auth] api login failed ip=${getClientIp(req)}`);
        return createErrorResponse("unauthenticated", "invalid password");
      }

      clearLoginFailures(req);
      console.log(`[auth] api login success ip=${getClientIp(req)}`);

      return createJsonResponse(
        { ok: true },
        {
          headers: {
            "Set-Cookie": buildSessionCookie(req, options.session),
          },
        },
      );
    }

    if (url.pathname.startsWith("/api/")) {
      if (!isAuthenticated(req, options.session)) {
        return unauthorizedResponse();
      }

      return handleDashboardApiRequest(req, url, options);
    }

    if (url.pathname === "/") {
      return redirectTo("/dashboard", req);
    }

    if (url.pathname === "/dashboard/login" && req.method === "POST") {
      return handleLoginForm(req, options);
    }

    if (url.pathname === "/dashboard" || url.pathname.startsWith("/dashboard/")) {
      if (isDashboardAssetPath(url.pathname)) {
        return serveDashboardAsset(url.pathname, dashboardBuilt);
      }

      if (options.session.password && !isAuthenticated(req, options.session) && url.pathname !== "/dashboard") {
        return redirectTo("/dashboard", req);
      }

      if (options.session.password && !isAuthenticated(req, options.session) && url.pathname === "/dashboard") {
        const errorMessage = url.searchParams.get("error") === "invalid"
          ? "Wrong password."
          : url.searchParams.get("error") === "rate-limit"
            ? "Too many attempts. Try again later."
            : undefined;
        return renderLoginResponse(options, errorMessage);
      }

      return serveDashboardAsset(url.pathname, dashboardBuilt);
    }

    if (req.method === "POST") {
      return handleWebhookRequest(req, options.getSettings(), (message, context) => options.sendConfiguredAlert(message, context));
    }

    return createErrorResponse("not_found", "not found");
  };
}

export function describeError(error: unknown) {
  if (error instanceof Error) return error.stack ?? `${error.name}: ${error.message}`;
  return String(error);
}

function describeRequest(req: Request) {
  try {
    return `${req.method} ${new URL(req.url).pathname}`;
  } catch {
    return `${req.method} ${req.url}`;
  }
}

export function createAppServer(options: AppServerOptions & { poolServer?: PoolServer }) {
  const handleRequest = createRequestHandler(options);

  return async (req: Request, server: unknown): Promise<Response> => {
    try {
      return await handleRequest(req, server);
    } catch (error) {
      console.error(`[server] unhandled error ${describeRequest(req)}: ${describeError(error)}`);
      return createErrorResponse("internal", "internal server error");
    }
  };
}
