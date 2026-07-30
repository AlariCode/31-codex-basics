import type { LoginInput, RegisterInput } from "@/features/auth/types";

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function validateLogin(input: LoginInput): string | null {
  if (!emailPattern.test(input.email.trim())) {
    return "Введите корректный email.";
  }
  if (input.password.length < 8) {
    return "Пароль должен содержать минимум 8 символов.";
  }
  return null;
}

export function validateRegistration(input: RegisterInput): string | null {
  if (input.name.trim().length === 0) {
    return "Введите ваше имя.";
  }
  return validateLogin(input);
}
