import type { AuthResponse, AuthUser, LoginInput, RegisterInput } from "@/features/auth/types";
import { APIError } from "@/features/api-error";

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

export async function uploadFile(accessToken: string, file: File): Promise<UploadedFile> {
  const formData = new FormData();
  formData.append("file", file);
  const response = await fetch(`${apiURL}/api/v1/uploads`, {
    method: "POST",
    headers: { Authorization: `Bearer ${accessToken}` },
    body: formData,
  });
  const payload = (await response.json().catch(() => null)) as UploadedFile | { error?: string } | null;
  if (!response.ok) {
    const message = payload && "error" in payload && payload.error ? payload.error : "Не удалось загрузить файл. Попробуйте ещё раз.";
    throw new AuthAPIError(message, response.status);
  }
  return payload as UploadedFile;
}

export class AuthAPIError extends APIError {
  constructor(message: string, status: number) {
    super(message, status);
    this.name = "AuthAPIError";
  }
}

export async function login(input: LoginInput): Promise<AuthResponse> {
  return request("/api/v1/auth/login", input);
}

export async function register(input: RegisterInput): Promise<AuthResponse> {
  return request("/api/v1/auth/register", input);
}

export async function refresh(): Promise<AuthResponse> {
  return request("/api/v1/auth/refresh");
}

export async function logout(): Promise<void> {
  const response = await fetch(`${apiURL}/api/v1/auth/logout`, {
    method: "POST",
    credentials: "include",
  });
  if (!response.ok) {
    throw new AuthAPIError("Не удалось завершить сессию.", response.status);
  }
}

export async function getProfile(accessToken: string): Promise<AuthUser> {
  return requestProfile("GET", accessToken);
}

export async function updateProfile(accessToken: string, name: string): Promise<AuthUser> {
  return requestProfile("PATCH", accessToken, { name });
}

export async function uploadAvatar(accessToken: string, file: File): Promise<AuthUser> {
  const formData = new FormData();
  formData.append("avatar", file);
  return requestProfile("POST", accessToken, formData, "/api/v1/profile/avatar");
}

async function request(path: string, body?: LoginInput | RegisterInput): Promise<AuthResponse> {
  const response = await fetch(`${apiURL}${path}`, {
    method: "POST",
    credentials: "include",
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });

  const payload = (await response.json().catch(() => null)) as AuthResponse | { error?: string } | null;
  if (!response.ok) {
    const message = payload && "error" in payload && payload.error ? payload.error : "Не удалось выполнить запрос. Попробуйте ещё раз.";
    throw new AuthAPIError(message, response.status);
  }
  return payload as AuthResponse;
}

async function requestProfile(method: "GET" | "PATCH" | "POST", accessToken: string, body?: { name: string } | FormData, path = "/api/v1/profile"): Promise<AuthUser> {
  const isFormData = body instanceof FormData;
  const response = await fetch(`${apiURL}${path}`, {
    method,
    headers: { Authorization: `Bearer ${accessToken}`, ...(body && !isFormData ? { "Content-Type": "application/json" } : {}) },
    body: body ? (isFormData ? body : JSON.stringify(body)) : undefined,
  });
  const payload = (await response.json().catch(() => null)) as AuthUser | { error?: string } | null;
  if (!response.ok) {
    const message = payload && "error" in payload && payload.error ? payload.error : "Не удалось выполнить запрос. Попробуйте ещё раз.";
    throw new AuthAPIError(message, response.status);
  }
  return payload as AuthUser;
}
