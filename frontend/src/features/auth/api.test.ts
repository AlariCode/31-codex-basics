import { afterEach, describe, expect, it, vi } from "vitest";

import { AuthAPIError, login, logout, refresh } from "@/features/auth/api";

describe("auth API client", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends login with JSON and browser credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ access_token: "access", token_type: "Bearer", expires_in: 86400, user: { id: "1", email: "person@example.com", name: "Person" } }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(login({ email: "person@example.com", password: "secure-pass" })).resolves.toMatchObject({ access_token: "access" });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/login",
      expect.objectContaining({ credentials: "include", method: "POST", body: JSON.stringify({ email: "person@example.com", password: "secure-pass" }) }),
    );
  });

  it("maps a failed refresh to an API error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: "invalid refresh token" }), { status: 401 })));

    await expect(refresh()).rejects.toEqual(expect.objectContaining<AuthAPIError>({ name: "AuthAPIError", status: 401 }));
  });

  it("sends logout with browser credentials", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await expect(logout()).resolves.toBeUndefined();
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/v1/auth/logout",
      expect.objectContaining({ credentials: "include", method: "POST" }),
    );
  });
});
