"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

import { useAuth } from "@/features/auth/auth-provider";
import { avatarURL } from "@/features/auth/api";

export function DashboardLayout({ children }: Readonly<{ children?: React.ReactNode }>) {
  const { logout, user } = useAuth();
  const router = useRouter();
  const [isProfileOpen, setIsProfileOpen] = useState(false);
  const [isLoggingOut, setIsLoggingOut] = useState(false);

  async function handleLogout() {
    setIsLoggingOut(true);
    try {
      await logout();
    } finally {
      router.replace("/login");
    }
  }

  return (
    <div className="min-h-screen bg-zinc-50 text-zinc-900">
      <header className="border-b border-zinc-200 bg-white">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-6">
          <span className="text-lg font-semibold tracking-tight">Uptime</span>
          <div className="flex items-center gap-2">
            <div className="relative">
              <button className="flex items-center gap-3 rounded-lg px-3 py-2 text-left hover:bg-zinc-100" type="button" onClick={() => setIsProfileOpen((isOpen) => !isOpen)} aria-expanded={isProfileOpen} aria-haspopup="menu" aria-label="Открыть меню профиля">
                {user?.avatar_url ? <img className="size-8 rounded-full object-cover" src={avatarURL(user.avatar_url) ?? undefined} alt="Аватар пользователя" /> : <span className="flex size-8 items-center justify-center rounded-full bg-blue-100 text-sm font-semibold text-blue-700">{user?.name.slice(0, 1).toUpperCase()}</span>}
                <span className="hidden text-sm font-medium sm:block">{user?.name}</span>
              </button>
              {isProfileOpen ? (
                <div className="absolute right-0 top-12 z-10 w-64 rounded-xl border border-zinc-200 bg-white p-2 shadow-lg" role="menu">
                  <div className="px-3 py-2">
                    <p className="text-sm font-medium">{user?.name}</p>
                    <p className="mt-1 truncate text-xs text-zinc-500">{user?.email}</p>
                  </div>
                  <Link className="block rounded-lg px-3 py-2 text-sm hover:bg-zinc-100" href="/profile" role="menuitem" onClick={() => setIsProfileOpen(false)}>Профиль</Link>
                </div>
              ) : null}
            </div>
            <button className="rounded-lg px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-60" type="button" onClick={handleLogout} disabled={isLoggingOut}>
              {isLoggingOut ? "Выходим…" : "Выйти"}
            </button>
          </div>
        </div>
      </header>
      {children}
    </div>
  );
}
