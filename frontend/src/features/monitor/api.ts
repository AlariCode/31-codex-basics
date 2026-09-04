import { requestWithAuth, type RefreshAccessToken } from "@/features/api-client";

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export type Monitor = {
  id: string;
  url: string;
  favicon_url: string;
  interval_seconds: number;
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
