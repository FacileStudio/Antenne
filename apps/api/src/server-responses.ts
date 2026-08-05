export type ErrorCode =
  | "invalid_argument"
  | "unauthenticated"
  | "permission_denied"
  | "not_found"
  | "already_exists"
  | "failed_precondition"
  | "rate_limited"
  | "resource_exhausted"
  | "internal";

const STATUS_BY_CODE: Record<ErrorCode, number> = {
  invalid_argument: 400,
  unauthenticated: 401,
  permission_denied: 403,
  not_found: 404,
  already_exists: 409,
  failed_precondition: 412,
  rate_limited: 429,
  resource_exhausted: 413,
  internal: 500,
};

export function createJsonResponse(body: unknown, init?: ResponseInit) {
  return new Response(JSON.stringify(body), {
    ...init,
    headers: {
      "Content-Type": "application/json",
      "Cache-Control": "private, no-store",
      Vary: "Cookie",
      ...(init?.headers ?? {}),
    },
  });
}

export function createErrorResponse(code: ErrorCode, message: string, init?: ResponseInit) {
  return createJsonResponse({ error: { code, message } }, {
    ...init,
    status: init?.status ?? STATUS_BY_CODE[code],
  });
}
