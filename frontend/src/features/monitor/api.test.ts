import { beforeEach, describe, expect, it, vi } from "vitest";

import { createMonitor } from "@/features/monitor/api";

describe("monitor API", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("creates a monitor with an interval expressed in seconds", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ id: "1", url: "https://example.com", interval_seconds: 420 }), { status: 201 }));

    await expect(createMonitor("token", { url: "https://example.com", interval_seconds: 420 })).resolves.toEqual({ id: "1", url: "https://example.com", interval_seconds: 420 });
    expect(fetchMock).toHaveBeenCalledWith("http://localhost:8080/api/v1/monitors", expect.objectContaining({ method: "POST", body: JSON.stringify({ url: "https://example.com", interval_seconds: 420 }) }));
  });
});
