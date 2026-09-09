import { requestWithAuth, type RefreshAccessToken } from "@/features/api-client";

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export type Monitor = {
  id: string;
  url: string;
  favicon_url: string;
  interval_seconds: number;
  last_checked_at: string | null;
  last_status: "pending" | "up" | "down" | "blocked";
  last_http_status: number | null;
  last_error: string;
};

export function faviconURL(path: string): string | null {
  if (!path) return null;
  return path.startsWith("http") ? path : `${apiURL}${path}`;
}

export async function listMonitors(refreshAccessToken: RefreshAccessToken, signal?: AbortSignal): Promise<Monitor[]> {
  return requestWithAuth<Monitor[]>({ method: "GET", url: "/api/v1/monitors", signal }, refreshAccessToken);
}

export async function createMonitor(refreshAccessToken: RefreshAccessToken, input: { url: string; interval_seconds: number }, signal?: AbortSignal): Promise<Monitor> {
  return requestWithAuth<Monitor>({
    method: "POST",
    url: "/api/v1/monitors",
    data: input,
    signal,
  }, refreshAccessToken);
}

export async function updateMonitor(refreshAccessToken: RefreshAccessToken, monitorID: string, input: { url: string; interval_seconds: number }, signal?: AbortSignal): Promise<Monitor> {
  return requestWithAuth<Monitor>({
    method: "PATCH",
    url: `/api/v1/monitors/${monitorID}`,
    data: input,
    signal,
  }, refreshAccessToken);
}

export async function deleteMonitor(refreshAccessToken: RefreshAccessToken, monitorID: string, signal?: AbortSignal): Promise<void> {
  await requestWithAuth<void>({
    method: "DELETE",
    url: `/api/v1/monitors/${monitorID}`,
    signal,
  }, refreshAccessToken);
}

export type MonitorPeriod = "1h" | "24h" | "7d" | "30d";

export type StatsPoint = {
  start: string;
  end: string;
  successes: number;
  failures: number;
  uptime_percent: number | null;
};

export type MonitorStats = {
  monitor_id: string;
  successes: number;
  failures: number;
  uptime_percent: number | null;
  points: StatsPoint[];
};

export type StatsResponse = {
  from: string;
  to: string;
  bucket_seconds: number;
  monitors: MonitorStats[];
};

export function getMonitorStats(refreshAccessToken: RefreshAccessToken, period: MonitorPeriod, signal?: AbortSignal): Promise<StatsResponse> {
  return requestWithAuth<StatsResponse>({ method: "GET", url: "/api/v1/monitors/stats", params: { period }, signal }, refreshAccessToken);
}
