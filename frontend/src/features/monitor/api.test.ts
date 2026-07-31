import { beforeEach, describe, expect, it, vi } from "vitest";

import { createMonitor, listMonitors } from "@/features/monitor/api";

describe("monitor API", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("creates a monitor with an interval expressed in seconds", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ id: "1", url: "https://example.com", interval_seconds: 420 }), { status: 201 }));

    await expect(createMonitor("token", async () => "token", { url: "https://example.com", interval_seconds: 420 })).resolves.toEqual({ id: "1", url: "https://example.com", interval_seconds: 420 });
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/api/v1/monitors", expect.objectContaining({ method: "POST", body: JSON.stringify({ url: "https://example.com", interval_seconds: 420 }), headers: { "Content-Type": "application/json", Authorization: "Bearer token" } }));
  });

  it("refreshes the access token once after an unauthorized response", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "expired" }), { status: 401 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([]), { status: 200 }));
    const refresh = vi.fn().mockResolvedValue("new-token");

    await expect(listMonitors("old-token", refresh)).resolves.toEqual([]);
    expect(refresh).toHaveBeenCalledOnce();
    expect(fetchMock).toHaveBeenLastCalledWith("http://localhost:8080/api/v1/monitors", expect.objectContaining({ headers: { Authorization: "Bearer new-token" } }));
  });

  it("returns an API error when refresh cannot restore the session", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ error: "expired" }), { status: 401 }));

    await expect(listMonitors("old-token", async () => null)).rejects.toEqual(expect.objectContaining({ name: "APIError", status: 401 }));
  });
});
