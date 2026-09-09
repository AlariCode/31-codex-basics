import { StrictMode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MonitorDashboard } from "@/features/monitor/monitor-dashboard";
import { createMonitor, getMonitorStats, deleteMonitor, listMonitors, updateMonitor } from "@/features/monitor/api";

const { refreshSession } = vi.hoisted(() => ({ refreshSession: vi.fn().mockResolvedValue("new-token") }));

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({ accessToken: "token", refreshSession }),
}));

vi.mock("@/features/monitor/api", () => ({
  getMonitorStats: vi.fn(),
  createMonitor: vi.fn(),
  deleteMonitor: vi.fn(),
  faviconURL: (path: string) => path ? `http://localhost:8080${path}` : null,
  listMonitors: vi.fn(),
  updateMonitor: vi.fn(),
}));

describe("MonitorDashboard", () => {

  it("preserves the previous graph when loading a new period fails", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{
      id: "1", url: "https://example.com", interval_seconds: 5, favicon_url: "",
      last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "",
    }]);
    vi.mocked(getMonitorStats).mockResolvedValueOnce({
      from: "2026-09-07T12:00:00Z", to: "2026-09-08T12:00:00Z", bucket_seconds: 900,
      monitors: [{ monitor_id: "1", successes: 9, failures: 1, uptime_percent: 90, points: [] }],
    }).mockRejectedValueOnce(new Error("offline"));
    render(<MonitorDashboard />);
    expect(await screen.findByText("90.0%")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Период графика"), { target: { value: "7d" } });
    expect(await screen.findByText(/Не удалось обновить график/)).toBeInTheDocument();
    expect(screen.getByText("90.0%")).toBeInTheDocument();
    expect(screen.getByText(/показан предыдущий график/)).toBeInTheDocument();
  });

  beforeEach(() => {
    vi.resetAllMocks();
    vi.mocked(getMonitorStats).mockResolvedValue({ from: "", to: "", bucket_seconds: 900, monitors: [] });
  });

  it("shows loading and then the empty state", async () => {
    vi.mocked(listMonitors).mockResolvedValue([]);

    render(<MonitorDashboard />);
    expect(screen.getByText("Загружаем сайты…")).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Пока нет сайтов. Добавьте первый сайт для мониторинга.")).toBeInTheDocument());
  });

  it("shows the downloaded favicon and replaces a failed image with the default site icon", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "/uploads/favicons/site.png", interval_seconds: 300 }]);

    render(<MonitorDashboard />);
    const favicon = await screen.findByTestId("monitor-favicon");
    expect(favicon).toHaveAttribute("src", "http://localhost:8080/uploads/favicons/site.png?site=https%3A%2F%2Fexample.com");
    fireEvent.error(favicon);

    expect(screen.getByTestId("default-site-icon")).toBeInTheDocument();
  });

  it("creates a monitor from the form and displays it", async () => {
    vi.mocked(listMonitors).mockResolvedValue([]);
    vi.mocked(createMonitor).mockResolvedValue({ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "", interval_seconds: 300 });

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("Пока нет сайтов. Добавьте первый сайт для мониторинга.")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("URL сайта"), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    expect(createMonitor).toHaveBeenCalledWith(expect.any(Function), { url: "https://example.com", interval_seconds: 5 });
  });

  it("displays a created monitor in React Strict Mode", async () => {
    vi.mocked(listMonitors).mockResolvedValue([]);
    vi.mocked(createMonitor).mockResolvedValue({ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "", interval_seconds: 5 });

    render(<StrictMode><MonitorDashboard /></StrictMode>);
    await waitFor(() => expect(screen.getByText("Пока нет сайтов. Добавьте первый сайт для мониторинга.")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Добавить сайт" }));
    fireEvent.change(screen.getByLabelText("URL сайта"), { target: { value: "https://example.com" } });
    fireEvent.click(screen.getByRole("button", { name: "Создать" }));

    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Создать" })).not.toBeInTheDocument();
  });

  it("edits a monitor using the prefilled form", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "", interval_seconds: 300 }]);
    vi.mocked(updateMonitor).mockResolvedValue({ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://updated.example", favicon_url: "", interval_seconds: 600 });

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Редактировать" }));
    expect(screen.getByLabelText("URL сайта")).toHaveValue("https://example.com");
    fireEvent.change(screen.getByLabelText("URL сайта"), { target: { value: "https://updated.example" } });
    fireEvent.change(screen.getByLabelText("Частота"), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить" }));

    await waitFor(() => expect(screen.getByText("https://updated.example")).toBeInTheDocument());
    expect(updateMonitor).toHaveBeenCalledWith(expect.any(Function), "1", { url: "https://updated.example", interval_seconds: 600 });
  });

  it("deletes a monitor after confirmation in the popup", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "", interval_seconds: 300 }]);
    vi.mocked(deleteMonitor).mockResolvedValue();

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Удалить" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("dialog").querySelector("button:last-child")!);

    await waitFor(() => expect(screen.queryByText("https://example.com")).not.toBeInTheDocument());
    expect(deleteMonitor).toHaveBeenCalledWith(expect.any(Function), "1");
  });

  it("closes the delete popup without deleting", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{ last_checked_at: null, last_status: "pending", last_http_status: null, last_error: "", id: "1", url: "https://example.com", favicon_url: "", interval_seconds: 300 }]);

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Удалить" }));
    fireEvent.click(screen.getByRole("dialog").querySelector("button:first-child")!);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(deleteMonitor).not.toHaveBeenCalled();
  });
});
