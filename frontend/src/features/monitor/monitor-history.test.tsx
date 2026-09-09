import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MonitorHistory, MonitorStatus } from "./monitor-history";
import type { Monitor } from "./api";

const monitor: Monitor = {
  id: "1", url: "https://example.com", interval_seconds: 5, favicon_url: "",
  last_checked_at: null, last_http_status: null, last_status: "pending", last_error: "",
};

describe("monitor observations", () => {
  it("distinguishes pending, stale, blocked and failed checks", () => {
    const { rerender } = render(<MonitorStatus monitor={monitor} />);
    expect(screen.getByText("Ожидаем проверку")).toBeInTheDocument();
    rerender(<MonitorStatus monitor={{ ...monitor, last_status: "up", last_checked_at: new Date(Date.now() - 21000).toISOString() }} />);
    expect(screen.getByText("Данные устарели")).toBeInTheDocument();
    rerender(<MonitorStatus monitor={{ ...monitor, last_status: "blocked" }} />);
    expect(screen.getByText("Адрес запрещён")).toBeInTheDocument();
    rerender(<MonitorStatus monitor={{ ...monitor, last_status: "down", last_checked_at: new Date().toISOString(), last_http_status: 500 }} />);
    expect(screen.getByText("Недоступен")).toBeInTheDocument();
    expect(screen.getByText(/HTTP 500/)).toBeInTheDocument();
  });

  it("shows missing data separately from failures and exposes counters on focus", () => {
    render(<MonitorHistory stats={{ monitor_id: "1", successes: 3, failures: 1, uptime_percent: 75, points: [
      { start: "2026-09-08T12:00:00Z", end: "2026-09-08T12:01:00Z", successes: 3, failures: 1, uptime_percent: 75 },
      { start: "2026-09-08T12:01:00Z", end: "2026-09-08T12:02:00Z", successes: 0, failures: 0, uptime_percent: null },
    ] }} />);
    expect(screen.getByText("75.0%")).toBeInTheDocument();
    const bars = screen.getAllByRole("button");
    expect(bars[1]).toHaveAccessibleName(/нет данных/);
    fireEvent.focus(bars[0]);
    expect(screen.getByRole("tooltip")).toHaveTextContent("успешно 3, неуспешно 1");
    fireEvent.keyDown(bars[0], { key: "ArrowRight" });
    expect(bars[1]).toHaveFocus();
  });
});
