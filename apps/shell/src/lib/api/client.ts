// Typed client for the ghostd daemon. In production ghostd serves the shell and
// injects a session token as window.__GHOST_TOKEN__; in dev, dev.sh writes
// the token and ghostd accepts it via the same mechanism (Vite proxies /api).
declare global {
  interface Window {
    __GHOST_TOKEN__?: string;
  }
}

let token: string | null = window.__GHOST_TOKEN__ ?? null;

export async function ensureToken(): Promise<string> {
  if (token) return token;
  // Dev fallback: ghostd exposes the token to localhost dev origins only.
  const res = await fetch('/api/v1/session/dev-token');
  if (!res.ok) throw new Error('no session token available');
  token = (await res.json()).token as string;
  return token;
}

export function getToken(): string | null {
  return token;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  await ensureToken();
  const res = await fetch(`/api/v1${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new ApiError(res.status, text || res.statusText);
  }
  const ct = res.headers.get('Content-Type') ?? '';
  return (ct.includes('application/json') ? res.json() : res.text()) as Promise<T>;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

export const api = {
  get: <T>(path: string) => request<T>('GET', path),
  post: <T>(path: string, body?: unknown) => request<T>('POST', path, body),
  put: <T>(path: string, body?: unknown) => request<T>('PUT', path, body),
  del: <T>(path: string) => request<T>('DELETE', path),
};

// ---- API types (mirrors packages/protocol/openapi.yaml) ----

export interface DirEntry {
  name: string;
  path: string;
  dir: boolean;
  size: number;
  modified: string; // RFC3339
  mime: string;
}

export interface SystemStatus {
  hostname: string;
  platform: string;
  wifi: { available: boolean; connected: boolean; ssid: string; signal: number };
  battery: { available: boolean; charging: boolean; percent: number };
  volume: { percent: number; muted: boolean };
}

export interface WifiNetwork {
  ssid: string;
  signal: number;
  secured: boolean;
  known: boolean;
  active: boolean;
}
