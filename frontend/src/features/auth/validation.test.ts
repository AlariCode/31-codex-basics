import { describe, expect, it } from "vitest";

import { validateLogin, validateRegistration } from "@/features/auth/validation";

describe("auth input validation", () => {
  it("rejects an invalid email and short password", () => {
    expect(validateLogin({ email: "invalid", password: "short" })).toBe("Введите корректный email.");
    expect(validateLogin({ email: "person@example.com", password: "short" })).toBe("Пароль должен содержать минимум 8 символов.");
  });

  it("requires a name during registration", () => {
    expect(validateRegistration({ email: "person@example.com", name: " ", password: "secure-pass" })).toBe("Введите ваше имя.");
    expect(validateRegistration({ email: "person@example.com", name: "Person", password: "secure-pass" })).toBeNull();
  });
});
