import { beforeEach, describe, expect, it, vi } from "vitest";
import axios from "axios";

import { apiClient, isRequestCanceled } from "@/features/api-client";
import { createMonitor, deleteMonitor, listMonitors, updateMonitor } from "@/features/monitor/api";

describe("monitor API", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("creates a monitor through the shared authenticated Axios client", async () => {
    const requestMock = vi.spyOn(apiClient, "request").mockResolvedValue({
      data: { id: "1", url: "https://example.com", interval_seconds: 420 },
    } as never);

    await expect(createMonitor(async () => "token", { url: "https://example.com", interval_seconds: 420 })).resolves.toEqual({ id: "1", url: "https://example.com", interval_seconds: 420 });
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({ method: "POST", url: "/api/v1/monitors", data: { url: "https://example.com", interval_seconds: 420 } }));
  });

  it("refreshes the access token once after an unauthorized response", async () => {
    const requestMock = vi.spyOn(apiClient, "request")
      .mockRejectedValueOnce(axiosError(401, { error: "expired" }))
      .mockResolvedValueOnce({ data: [] } as never);
    const refresh = vi.fn().mockResolvedValue("new-token");

    await expect(listMonitors(refresh)).resolves.toEqual([]);
    expect(refresh).toHaveBeenCalledOnce();
    expect(requestMock).toHaveBeenCalledTimes(2);
  });

  it("updates a monitor through the authenticated PATCH endpoint", async () => {
    const requestMock = vi.spyOn(apiClient, "request").mockResolvedValue({
      data: { id: "1", url: "https://updated.example", interval_seconds: 600 },
    } as never);

    await expect(updateMonitor(async () => "token", "1", { url: "https://updated.example", interval_seconds: 600 })).resolves.toEqual({ id: "1", url: "https://updated.example", interval_seconds: 600 });
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({ method: "PATCH", url: "/api/v1/monitors/1" }));
  });

  it("deletes a monitor through the authenticated DELETE endpoint", async () => {
    const requestMock = vi.spyOn(apiClient, "request").mockResolvedValue({ data: undefined } as never);

    await expect(deleteMonitor(async () => "token", "1")).resolves.toBeUndefined();
    expect(requestMock).toHaveBeenCalledWith(expect.objectContaining({ method: "DELETE", url: "/api/v1/monitors/1" }));
  });

  it("returns an API error when refresh cannot restore the session", async () => {
    vi.spyOn(apiClient, "request").mockRejectedValue(axiosError(401, { error: "expired" }));

    await expect(listMonitors(async () => null)).rejects.toEqual(expect.objectContaining({ name: "APIError", status: 401 }));
  });

  it("preserves Axios cancellation errors", async () => {
    const cancellation = new axios.CanceledError("request canceled");
    vi.spyOn(apiClient, "request").mockRejectedValue(cancellation);

    await expect(listMonitors(async () => "new-token")).rejects.toBe(cancellation);
    expect(isRequestCanceled(cancellation)).toBe(true);
  });
});

function axiosError(status: number, data: { error: string }): Error {
  return Object.assign(new Error("request failed"), { isAxiosError: true, response: { status, data } });
}
