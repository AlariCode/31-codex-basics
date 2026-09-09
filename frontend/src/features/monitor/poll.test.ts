import { afterEach, describe, expect, it, vi } from "vitest";
import { startPolling } from "./poll";

afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); });

describe("monitor polling", () => {
  it("does not overlap requests and cancels on hidden/unmount", async () => {
    vi.useFakeTimers();
    const hidden = vi.spyOn(document, "hidden", "get").mockReturnValue(false);
    let finish = () => {};
    const load = vi.fn<(signal: AbortSignal) => Promise<void>>(() => new Promise<void>((resolve) => { finish = resolve; }));
    const stop = startPolling(load, 5000);
    await vi.advanceTimersByTimeAsync(15000);
    expect(load).toHaveBeenCalledTimes(1);
    finish();
    await vi.advanceTimersByTimeAsync(5000);
    expect(load).toHaveBeenCalledTimes(2);
    hidden.mockReturnValue(true);
    document.dispatchEvent(new Event("visibilitychange"));
    expect(load.mock.calls[1][0].aborted).toBe(true);
    await vi.advanceTimersByTimeAsync(15000);
    expect(load).toHaveBeenCalledTimes(2);
    hidden.mockReturnValue(false);
    document.dispatchEvent(new Event("visibilitychange"));
    expect(load).toHaveBeenCalledTimes(3);
    stop();
    expect(load.mock.calls[2][0].aborted).toBe(true);
  });
});
