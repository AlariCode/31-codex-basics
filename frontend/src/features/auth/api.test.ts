import { afterEach, describe, expect, it, vi } from "vitest";

import { apiClient } from "@/features/api-client";
import { login, logout, refresh } from "@/features/auth/api";

describe("auth API client", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("sends login through the shared Axios client", async () => {
    const requestMock = vi.spyOn(apiClient, "request").mockResolvedValue({
      data: { access_token: "access", token_type: "Bearer", expires_in: 86400, user: { id: "1", email: "person@example.com", name: "Person" } },
    } as never);

    await expect(login({ email: "person@example.com", password: "secure-pass" })).resolves.toMatchObject({ access_token: "access" });
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({ method: "POST", url: "/api/v1/auth/login", data: { email: "person@example.com", password: "secure-pass" } }));
  });

  it("maps a failed refresh to an API error", async () => {
    vi.spyOn(apiClient, "request").mockRejectedValue(axiosError(401, { error: "invalid refresh token" }));

    await expect(refresh()).rejects.toEqual(expect.objectContaining({ name: "AuthAPIError", status: 401 }));
  });

  it("sends logout through the shared Axios client", async () => {
    const requestMock = vi.spyOn(apiClient, "request").mockResolvedValue({ data: undefined, status: 204 } as never);

    await expect(logout()).resolves.toBeUndefined();
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({ method: "POST", url: "/api/v1/auth/logout" }));
  });
});

function axiosError(status: number, data: { error: string }): Error {
  return Object.assign(new Error("request failed"), { isAxiosError: true, response: { status, data } });
}
