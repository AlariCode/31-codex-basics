import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { MonitorDashboard } from "@/features/monitor/monitor-dashboard";
import { createMonitor, deleteMonitor, listMonitors, updateMonitor } from "@/features/monitor/api";

const { refreshSession } = vi.hoisted(() => ({ refreshSession: vi.fn().mockResolvedValue("new-token") }));

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({ accessToken: "token", refreshSession }),
}));

vi.mock("@/features/monitor/api", () => ({
  createMonitor: vi.fn(),
  deleteMonitor: vi.fn(),
  listMonitors: vi.fn(),
  updateMonitor: vi.fn(),
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
    expect(createMonitor).toHaveBeenCalledWith(expect.any(Function), { url: "https://example.com", interval_seconds: 5 });
  });

  it("edits a monitor using the prefilled form", async () => {
    vi.mocked(listMonitors).mockResolvedValue([{ id: "1", url: "https://example.com", interval_seconds: 300 }]);
    vi.mocked(updateMonitor).mockResolvedValue({ id: "1", url: "https://updated.example", interval_seconds: 600 });

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
    vi.mocked(listMonitors).mockResolvedValue([{ id: "1", url: "https://example.com", interval_seconds: 300 }]);
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
    vi.mocked(listMonitors).mockResolvedValue([{ id: "1", url: "https://example.com", interval_seconds: 300 }]);

    render(<MonitorDashboard />);
    await waitFor(() => expect(screen.getByText("https://example.com")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Удалить" }));
    fireEvent.click(screen.getByRole("dialog").querySelector("button:first-child")!);

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(deleteMonitor).not.toHaveBeenCalled();
  });
});
