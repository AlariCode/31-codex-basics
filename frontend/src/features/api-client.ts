import axios, { type AxiosError, type AxiosRequestConfig } from "axios";

import { APIError } from "@/features/api-error";

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export type RefreshAccessToken = () => Promise<string | null>;

export const apiClient = axios.create({
  baseURL: apiURL,
  withCredentials: true,
});

let accessToken: string | null = null;

apiClient.interceptors.request.use((config) => {
  if (accessToken) {
    config.headers = config.headers ?? {};
    config.headers.Authorization = `Bearer ${accessToken}`;
  }
  return config;
});

export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  try {
    return (await apiClient.request<T>(config)).data;
  } catch (error) {
    throw toAPIError(error);
  }
}

export async function requestWithAuth<T>(config: AxiosRequestConfig, refreshAccessToken: RefreshAccessToken): Promise<T> {
  try {
    return (await apiClient.request<T>(config)).data;
  } catch (error) {
    if (!isUnauthorized(error)) {
      throw toAPIError(error);
    }
    let refreshedToken: string | null;
    try {
      refreshedToken = await refreshAccessToken();
    } catch {
      throw toAPIError(error);
    }
    if (!refreshedToken) {
      throw toAPIError(error);
    }
    try {
      return (await apiClient.request<T>(config)).data;
    } catch (retryError) {
      throw toAPIError(retryError);
    }
  }
}

function isUnauthorized(error: unknown): boolean {
  return isAxiosError(error) && error.response?.status === 401;
}

function isAxiosError(error: unknown): error is AxiosError<{ error?: string }> {
  return axios.isAxiosError(error);
}

function toAPIError(error: unknown): APIError {
  if (isAxiosError(error)) {
    const message = error.response?.data?.error ?? "Не удалось выполнить запрос.";
    return new APIError(message, error.response?.status ?? 0);
  }
  return new APIError("Не удалось выполнить запрос.", 0);
}
