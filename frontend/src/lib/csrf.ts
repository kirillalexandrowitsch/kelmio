export const CSRF_HEADER_NAME = "X-CSRF-Token";
export const CSRF_TOKEN_PATH = "/api/v1/auth/csrf-token";

const unsafeMethods = new Set(["POST", "PATCH", "PUT", "DELETE"]);

export function requestNeedsCSRF(path: string, method = "GET") {
  const normalizedMethod = method.toUpperCase();
  return (
    unsafeMethods.has(normalizedMethod) &&
    path !== "/api/v1/auth/login" &&
    !isInviteAcceptPath(path) &&
    path !== CSRF_TOKEN_PATH
  );
}

export function isInviteAcceptPath(path: string) {
  return path.startsWith("/api/v1/auth/invites/") && path.endsWith("/accept");
}

export function isCSRFError(status: number, code: unknown) {
  return (
    status === 403 &&
    (code === "csrf_token_required" || code === "invalid_csrf_token")
  );
}
