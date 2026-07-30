import { AuthAPIError } from "@/features/auth/api";

export type Monitor = {
  id: string;
  url: string;
  interval_seconds: number;
};

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export async function listMonitors(accessToken: string): Promise<Monitor[]> {
  const response = await fetch(`${apiURL}/api/v1/monitors`, { headers: { Authorization: `Bearer ${accessToken}` } });
  return parseResponse(response);
}

export async function createMonitor(accessToken: string, input: { url: string; interval_seconds: number }): Promise<Monitor> {
  const response = await fetch(`${apiURL}/api/v1/monitors`, {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}`, "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  return parseResponse(response);
}

async function parseResponse<T>(response: Response): Promise<T> {
  const payload = (await response.json().catch(() => null)) as T | { error?: string } | null;
  if (!response.ok) {
    const message = payload && typeof payload === "object" && "error" in payload && payload.error ? payload.error : "Не удалось выполнить запрос.";
    throw new AuthAPIError(message, response.status);
  }
  return payload as T;
}
