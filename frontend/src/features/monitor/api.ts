import { requestWithAccessToken, type RefreshAccessToken } from "@/features/api-client";

export type Monitor = {
  id: string;
  url: string;
  interval_seconds: number;
};

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export async function listMonitors(accessToken: string, refreshAccessToken: RefreshAccessToken, signal?: AbortSignal): Promise<Monitor[]> {
  return requestWithAccessToken<Monitor[]>(`${apiURL}/api/v1/monitors`, accessToken, refreshAccessToken, { signal });
}

export async function createMonitor(accessToken: string, refreshAccessToken: RefreshAccessToken, input: { url: string; interval_seconds: number }, signal?: AbortSignal): Promise<Monitor> {
  return requestWithAccessToken<Monitor>(`${apiURL}/api/v1/monitors`, accessToken, refreshAccessToken, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
    signal,
  });
}
