import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { AuthForm } from "@/features/auth/auth-form";

const mocks = vi.hoisted(() => ({
  login: vi.fn(),
  register: vi.fn(),
  replace: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: mocks.replace }),
}));

vi.mock("@/features/auth/auth-provider", () => ({
  useAuth: () => ({ login: mocks.login, register: mocks.register }),
}));

describe("AuthForm", () => {
  beforeEach(() => {
    mocks.login.mockReset();
    mocks.register.mockReset();
    mocks.replace.mockReset();
  });

  it("shows validation feedback before making a login request", async () => {
    const user = userEvent.setup();
    render(<AuthForm mode="login" />);

    await user.click(screen.getByRole("button", { name: "Войти" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Введите корректный email.");
    expect(mocks.login).not.toHaveBeenCalled();
  });

  it("submits valid login credentials and redirects home", async () => {
    mocks.login.mockResolvedValue(undefined);
    const user = userEvent.setup();
    render(<AuthForm mode="login" />);

    await user.type(screen.getByLabelText("Email"), "person@example.com");
    await user.type(screen.getByLabelText("Пароль"), "secure-pass");
    await user.click(screen.getByRole("button", { name: "Войти" }));

    expect(mocks.login).toHaveBeenCalledWith({ email: "person@example.com", password: "secure-pass" });
    expect(mocks.replace).toHaveBeenCalledWith("/");
  });
});
