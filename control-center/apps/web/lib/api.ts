import type { Problem } from "./types";

export class ApiError extends Error {
  code: string;
  remediation: string;

  constructor(body: Problem) {
    super(body.message || "Permintaan gagal");
    this.code = body.code || "ERROR";
    this.remediation = body.remediation || "";
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    cache: "no-store",
    headers: { Accept: "application/json", ...(init?.headers || {}) },
  });
  const text = await res.text();
  const body = text ? (JSON.parse(text) as Problem) : {};
  if (res.status === 401 && !path.endsWith("/session")) {
    if (typeof window !== "undefined") window.location.assign("/login");
    throw new ApiError(body);
  }
  if (!res.ok) throw new ApiError(body);
  return body as T;
}

export function csrfToken(): string {
  const match = document.cookie.match(/(?:^|; )hcc_csrf=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : "";
}

export function idempotencyKey(): string {
  return crypto.randomUUID().replace(/-/g, "");
}

export function when(value: string | null | undefined): string {
  if (!value) return "belum ada";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString("id-ID", { timeZone: "Asia/Jakarta", hour12: false });
}

export function bytes(n: number): string {
  if (!n) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  let v = n;
  let i = 0;
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024;
    i++;
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}
