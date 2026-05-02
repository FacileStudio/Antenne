import { createHmac, randomBytes, timingSafeEqual } from "node:crypto";

import type { DashboardSessionConfig } from "./types";

const dashboardSessionMaxAgeSeconds = 60 * 60 * 24 * 30;
const sessionTokenSecret = process.env.NOOK_SESSION_SECRET ?? randomBytes(32).toString("hex");
const loginAttempts = new Map<string, { count: number; firstFailedAt: number; blockedUntil: number }>();
const loginAttemptWindowMs = 10 * 60 * 1000;
const loginAttemptBlockMs = 10 * 60 * 1000;
const maxLoginAttempts = 5;

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

function createSessionToken(session: DashboardSessionConfig) {
  const issuedAt = Date.now().toString();
  const payload = `${session.username}:${issuedAt}:${session.password ?? ""}`;
  const signature = createHmac("sha256", sessionTokenSecret).update(payload).digest("base64url");
  return `${issuedAt}.${signature}`;
}

function getClientAddress(req: Request) {
  const forwarded = req.headers.get("x-forwarded-for");
  if (forwarded) {
    return forwarded.split(",")[0]?.trim() ?? "unknown";
  }

  return req.headers.get("cf-connecting-ip")
    ?? req.headers.get("x-real-ip")
    ?? "unknown";
}

export function isLoginRateLimited(req: Request) {
  const now = Date.now();
  const attempt = loginAttempts.get(getClientAddress(req));
  if (!attempt) {
    return false;
  }

  if (attempt.blockedUntil > now) {
    return true;
  }

  if (attempt.firstFailedAt + loginAttemptWindowMs < now) {
    loginAttempts.delete(getClientAddress(req));
  }

  return false;
}

export function recordLoginFailure(req: Request) {
  const key = getClientAddress(req);
  const now = Date.now();
  const current = loginAttempts.get(key);

  if (!current || current.firstFailedAt + loginAttemptWindowMs < now) {
    loginAttempts.set(key, {
      count: 1,
      firstFailedAt: now,
      blockedUntil: 0,
    });
    return;
  }

  current.count += 1;
  if (current.count >= maxLoginAttempts) {
    current.blockedUntil = now + loginAttemptBlockMs;
  }
}

export function clearLoginFailures(req: Request) {
  loginAttempts.delete(getClientAddress(req));
}

export function createDashboardSessionConfig(): DashboardSessionConfig {
  const password = process.env.NOOK_ADMIN_PASSWORD ?? null;

  return {
    enabled: true,
    username: process.env.NOOK_ADMIN_USERNAME ?? "admin",
    password,
    cookieName: "nook_session",
  };
}

export function buildSessionCookie(req: Request, session: DashboardSessionConfig) {
  const flags = [
    `${session.cookieName}=${encodeURIComponent(createSessionToken(session))}`,
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

export function buildExpiredSessionCookie(req: Request, session: DashboardSessionConfig) {
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

export function isAuthenticated(req: Request, session: DashboardSessionConfig) {
  if (!session.enabled || !session.password) {
    return true;
  }

  const cookie = getCookie(req, session.cookieName);
  if (!cookie) {
    return false;
  }

  const [issuedAt, signature] = cookie.split(".");
  if (!issuedAt || !signature) {
    return false;
  }

  const issuedAtNumber = Number(issuedAt);
  if (!Number.isFinite(issuedAtNumber) || issuedAtNumber + dashboardSessionMaxAgeSeconds * 1000 < Date.now()) {
    return false;
  }

  const payload = `${session.username}:${issuedAt}:${session.password}`;
  const expectedSignature = createHmac("sha256", sessionTokenSecret).update(payload).digest("base64url");
  return safeEquals(signature, expectedSignature);
}

export function isValidSessionPassword(password: string | null | undefined, session: DashboardSessionConfig) {
  if (!session.password || !password) {
    return false;
  }

  return safeEquals(password, session.password);
}
