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
import { createJsonResponse } from "./server-responses";
import { handleWebhookRequest } from "./webhooks";

function unauthorizedResponse() {
  return createJsonResponse({ error: "unauthorized" }, { status: 401 });
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
    return redirectTo("/dashboard?error=rate-limit", req, { status: 303 });
  }

  const formData = await req.formData();
  const password = formData.get("password");

  if (!isValidSessionPassword(typeof password === "string" ? password : null, options.session)) {
    recordLoginFailure(req);
    return redirectTo("/dashboard?error=invalid", req, { status: 303 });
  }

  clearLoginFailures(req);

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

export function createAppServer(options: AppServerOptions) {
  return async (req: Request): Promise<Response> => {
    const url = new URL(req.url);
    const dashboardBuilt = options.getRuntimeStatus().dashboardBuilt;

    if (url.pathname === "/health") {
      return new Response("ok", { status: 200 });
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
        return createJsonResponse({ error: "too many attempts" }, { status: 429 });
      }

      let body: { password?: string };
      try {
        body = await req.json() as { password?: string };
      } catch {
        return createJsonResponse({ error: "Invalid JSON body." }, { status: 400 });
      }

      if (!isValidSessionPassword(body.password, options.session)) {
        recordLoginFailure(req);
        return createJsonResponse({ error: "invalid password" }, { status: 401 });
      }

      clearLoginFailures(req);

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

    return new Response("not found", { status: 404 });
  };
}
