import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MonitorDashboard } from "@/features/monitor/monitor-dashboard";
import { createMonitor, listMonitors } from "@/features/monitor/api";

const { refreshSession } = vi.hoisted(() => ({ refreshSession: vi.fn().mockResolvedValue("new-token") }));

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({ accessToken: "token", refreshSession }),
}));

vi.mock("@/features/monitor/api", () => ({
  createMonitor: vi.fn(),
  listMonitors: vi.fn(),
}));

describe("MonitorDashboard", () => {
  beforeEach(() => vi.resetAllMocks());

  it("shows loading and then the empty state", async () => {
    vi.mocked(listMonitors).mockResolvedValue([]);

    render(<MonitorDashboard />);
    expect(screen.getByText("Загружаем сайты…")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Пока нет сайтов. Добавьте первый сайт для мониторинга.")).toBeInTheDocument());
  });

  it("creates a monitor from the form and displays it", async () => {
    vi.mocked(listMonitors).mockResolvedValue([]);
    vi.mocked(createMonitor).mockResolvedValue({ id: "1", url: "https://example.com", interval_seconds: 300 });

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("Пока нет сайтов. Добавьте первый сайт для мониторинга.")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("URL сайта"), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    expect(createMonitor).toHaveBeenCalledWith("token", expect.any(Function), { url: "https://example.com", interval_seconds: 5 });
  });
});
