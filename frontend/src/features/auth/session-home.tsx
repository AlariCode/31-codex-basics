"use client";

import Link from "next/link";

import { useAuth } from "@/features/auth/auth-provider";
import { DashboardLayout } from "@/features/auth/dashboard-layout";

export function SessionHome() {
  const { status, user } = useAuth();

  if (status === "loading") {
    return <main className="flex min-h-screen items-center justify-center bg-zinc-50 text-sm text-zinc-600">Восстанавливаем сессию…</main>;
  }

  if (status === "authenticated" && user) {
    return <DashboardLayout />;
  }

  return (
    <main className="flex min-h-screen items-center justify-center bg-zinc-50 px-6 py-12 text-zinc-900">
      <section className="w-full max-w-xl rounded-2xl border border-zinc-200 bg-white p-8 shadow-sm">
        <p className="text-sm font-medium text-blue-700">Uptime</p>
        <h1 className="mt-2 text-3xl font-semibold tracking-tight">Следите за доступностью сервисов</h1>
        <p className="mt-3 max-w-lg leading-7 text-zinc-600">Войдите в аккаунт или создайте новый, чтобы начать работу.</p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <Link className="rounded-lg bg-blue-700 px-4 py-2.5 text-center font-medium text-white hover:bg-blue-800" href="/login">Войти</Link>
          <Link className="rounded-lg border border-zinc-300 px-4 py-2.5 text-center font-medium hover:bg-zinc-50" href="/register">Создать аккаунт</Link>
        </div>
      </section>
    </main>
  );
}
