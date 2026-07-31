import type { AxiosRequestConfig } from "axios";

import { request, requestWithAuth, type RefreshAccessToken } from "@/features/api-client";
import { APIError } from "@/features/api-error";
import type { AuthResponse, AuthUser, LoginInput, RegisterInput } from "@/features/auth/types";

const apiURL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

export function avatarURL(path: string | null): string | null {
  if (!path) return null;
  return path.startsWith("http") ? path : `${apiURL}${path}`;
}

export type UploadedFile = {
  url: string;
  filename: string;
  content_type: string;
  size: number;
};

export async function uploadFile(file: File, refreshAccessToken: RefreshAccessToken): Promise<UploadedFile> {
  const formData = new FormData();
  formData.append("file", file);
  return requestAuth<UploadedFile>({ method: "POST", url: "/api/v1/uploads", data: formData }, refreshAccessToken);
}

export class AuthAPIError extends APIError {
  constructor(message: string, status: number) {
    super(message, status);
    this.name = "AuthAPIError";
  }
}

export async function login(input: LoginInput): Promise<AuthResponse> {
  return requestAuthResponse({ method: "POST", url: "/api/v1/auth/login", data: input });
}

export async function register(input: RegisterInput): Promise<AuthResponse> {
  return requestAuthResponse({ method: "POST", url: "/api/v1/auth/register", data: input });
}

export async function refresh(): Promise<AuthResponse> {
  return requestAuthResponse({ method: "POST", url: "/api/v1/auth/refresh" });
}

export async function logout(): Promise<void> {
  try {
    await request<void>({ method: "POST", url: "/api/v1/auth/logout" });
  } catch {
    throw new AuthAPIError("Не удалось завершить сессию.", 0);
  }
}

export async function getProfile(refreshAccessToken: RefreshAccessToken): Promise<AuthUser> {
  return requestAuth<AuthUser>({ method: "GET", url: "/api/v1/profile" }, refreshAccessToken);
}

export async function updateProfile(name: string, refreshAccessToken: RefreshAccessToken): Promise<AuthUser> {
  return requestAuth<AuthUser>({ method: "PATCH", url: "/api/v1/profile", data: { name } }, refreshAccessToken);
}

export async function uploadAvatar(file: File, refreshAccessToken: RefreshAccessToken): Promise<AuthUser> {
  const formData = new FormData();
  formData.append("avatar", file);
  return requestAuth<AuthUser>({ method: "POST", url: "/api/v1/profile/avatar", data: formData }, refreshAccessToken);
}

async function requestAuth<T>(config: AxiosRequestConfig, refreshAccessToken: RefreshAccessToken): Promise<T> {
  try {
    return await requestWithAuth<T>(config, refreshAccessToken);
  } catch (error) {
    throw toAuthError(error);
  }
}

async function requestAuthResponse(config: AxiosRequestConfig): Promise<AuthResponse> {
  try {
    return await request<AuthResponse>(config);
  } catch (error) {
    throw toAuthError(error);
  }
}

function toAuthError(error: unknown): AuthAPIError {
  if (error instanceof APIError) {
    return new AuthAPIError(error.message, error.status);
  }
  return new AuthAPIError("Не удалось выполнить запрос. Попробуйте ещё раз.", 0);
}
