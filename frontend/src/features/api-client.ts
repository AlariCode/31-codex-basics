import { APIError } from "@/features/api-error";

export type RefreshAccessToken = () => Promise<string | null>;

export async function requestWithAccessToken<T>(
  url: string,
  accessToken: string,
  refreshAccessToken: RefreshAccessToken,
  init: RequestInit = {},
): Promise<T> {
  let response = await send(url, accessToken, init);
  if (response.status === 401) {
    const refreshedToken = await refreshAccessToken();
    if (refreshedToken) {
      response = await send(url, refreshedToken, init);
    }
  }
  return parseResponse<T>(response);
}

async function send(url: string, accessToken: string, init: RequestInit): Promise<Response> {
  return fetch(url, {
    ...init,
    headers: { ...init.headers, Authorization: `Bearer ${accessToken}` },
  });
}

async function parseResponse<T>(response: Response): Promise<T> {
  const payload = (await response.json().catch(() => null)) as T | { error?: string } | null;
  if (!response.ok) {
    const message = payload && typeof payload === "object" && "error" in payload && payload.error ? payload.error : "Не удалось выполнить запрос.";
    throw new APIError(message, response.status);
  }
  return payload as T;
}
