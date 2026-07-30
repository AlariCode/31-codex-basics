import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DashboardLayout } from "@/features/auth/dashboard-layout";

const mocks = vi.hoisted(() => ({
  logout: vi.fn(),
  replace: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mocks.replace }),
}));

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({
    logout: mocks.logout,
    user: { id: "1", name: "Person", email: "person@example.com" },
  }),
}));

describe("DashboardLayout", () => {
  beforeEach(() => {
    mocks.logout.mockReset();
    mocks.replace.mockReset();
  });

  it("opens the profile menu and logs the user out", async () => {
    mocks.logout.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<DashboardLayout><p>Dashboard</p></DashboardLayout>);

    await user.click(screen.getByRole("button", { name: "Открыть меню профиля" }));
    expect(screen.getByText("person@example.com")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Выйти" }));

    expect(mocks.logout).toHaveBeenCalledOnce();
    expect(mocks.replace).toHaveBeenCalledWith("/login");
  });
});
