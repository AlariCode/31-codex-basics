"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";

import { AuthAPIError } from "@/features/auth/api";
import { useAuth } from "@/features/auth/auth-provider";
import { validateLogin, validateRegistration } from "@/features/auth/validation";

type AuthFormProps = {
  mode: "login" | "register";
};

export function AuthForm({ mode }: AuthFormProps) {
  const { login, register } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const isRegistration = mode === "register";

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const loginInput = { email, password };
    const validationError = isRegistration ? validateRegistration({ ...loginInput, name }) : validateLogin(loginInput);
    if (validationError) {
      setError(validationError);
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      if (isRegistration) {
        await register({ ...loginInput, name });
      } else {
        await login(loginInput);
      }
      router.replace("/");
    } catch (requestError) {
      setError(messageForError(requestError));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-50 px-6 py-12 text-zinc-900">
      <section className="w-full max-w-md rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm">
        <p className="text-sm font-medium text-blue-700">Uptime</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">{isRegistration ? "Создайте аккаунт" : "Войдите в аккаунт"}</h1>
        <p className="mt-3 text-sm leading-6 text-zinc-600">
          {isRegistration ? "Создайте учётную запись, чтобы начать отслеживать сервисы." : "Используйте email и пароль для продолжения."}
        </p>

        <form className="mt-8 space-y-5" onSubmit={handleSubmit} noValidate>
          {isRegistration ? (
            <label className="block text-sm font-medium">
              Имя
              <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100" value={name} onChange={(event) => setName(event.target.value)} autoComplete="name" disabled={isSubmitting} />
            </label>
          ) : null}
          <label className="block text-sm font-medium">
            Email
            <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100" type="email" value={email} onChange={(event) => setEmail(event.target.value)} autoComplete="email" disabled={isSubmitting} />
          </label>
          <label className="block text-sm font-medium">
            Пароль
            <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100" type="password" value={password} onChange={(event) => setPassword(event.target.value)} autoComplete={isRegistration ? "new-password" : "current-password"} disabled={isSubmitting} />
          </label>
          {error ? <p className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700" role="alert">{error}</p> : null}
          <button className="w-full rounded-lg bg-blue-700 px-4 py-2.5 font-medium text-white transition hover:bg-blue-800 disabled:cursor-not-allowed disabled:opacity-60" type="submit" disabled={isSubmitting}>
            {isSubmitting ? "Подождите…" : isRegistration ? "Создать аккаунт" : "Войти"}
          </button>
        </form>

        <p className="mt-6 text-center text-sm text-zinc-600">
          {isRegistration ? "Уже есть аккаунт?" : "Нет аккаунта?"}{" "}
          <Link className="font-medium text-blue-700 hover:underline" href={isRegistration ? "/login" : "/register"}>
            {isRegistration ? "Войти" : "Зарегистрироваться"}
          </Link>
        </p>
      </section>
    </main>
  );
}

function messageForError(error: unknown): string {
  if (!(error instanceof AuthAPIError)) {
    return "Не удалось связаться с сервером. Попробуйте ещё раз.";
  }
  if (error.status === 409) {
    return "Этот email уже зарегистрирован.";
  }
  if (error.status === 401) {
    return "Неверный email или пароль.";
  }
  if (error.status === 400) {
    return "Проверьте введённые данные.";
  }
  return "Не удалось выполнить запрос. Попробуйте ещё раз.";
}
