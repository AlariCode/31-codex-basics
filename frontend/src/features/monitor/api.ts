import { requestWithAuth, type RefreshAccessToken } from "@/features/api-client";

export type Monitor = {
  id: string;
  url: string;
  interval_seconds: number;
};

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
