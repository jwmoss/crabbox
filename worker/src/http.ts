export function json(data: unknown, init: ResponseInit = {}): Response {
  const headers = new Headers(init.headers);
  headers.set("content-type", "application/json; charset=utf-8");
  return new Response(JSON.stringify(redactStackTraceFields(data)), { ...init, headers });
}

export function text(message: string, status = 200): Response {
  return new Response(message, {
    status,
    headers: { "content-type": "text/plain; charset=utf-8" },
  });
}

export async function readJson<T>(request: Request): Promise<T> {
  const value = (await request.json()) as unknown;
  return value as T;
}

export function bearerToken(request: Request): string {
  const header = request.headers.get("authorization") ?? "";
  const [scheme, token] = header.split(" ", 2);
  if (scheme?.toLowerCase() !== "bearer" || !token) {
    return "";
  }
  return token;
}

export function requestOwner(request: Request): string {
  return request.headers.get("x-crabbox-owner") ?? "unknown";
}

export function pathParts(request: Request): string[] {
  return new URL(request.url).pathname.split("/").filter(Boolean);
}

export function errorMessage(error: unknown): string {
  return firstLine(error instanceof Error ? error.message : String(error));
}

function redactStackTraceFields(value: unknown, seen = new WeakSet<object>()): unknown {
  if (value instanceof Error) {
    return { name: value.name, message: firstLine(value.message) };
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  if (seen.has(value)) {
    return "[Circular]";
  }
  seen.add(value);
  const jsonValue = (value as { toJSON?: () => unknown }).toJSON;
  if (typeof jsonValue === "function") {
    const next = jsonValue.call(value);
    if (next !== value) {
      const output = redactStackTraceFields(next, seen);
      seen.delete(value);
      return output;
    }
  }
  if (Array.isArray(value)) {
    const output = value.map((item) => redactStackTraceFields(item, seen));
    seen.delete(value);
    return output;
  }
  const output: Record<string, unknown> = {};
  for (const [key, item] of Object.entries(value)) {
    if (key !== "stack" && typeof item !== "function") {
      output[key] = redactStackTraceFields(item, seen);
    }
  }
  seen.delete(value);
  return output;
}

function firstLine(value: string): string {
  const index = value.indexOf("\n");
  return index >= 0 ? value.slice(0, index) : value;
}
