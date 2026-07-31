"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";

import * as authAPI from "@/features/auth/api";
import type { AuthResponse, AuthUser, LoginInput, RegisterInput } from "@/features/auth/types";

type SessionStatus = "loading" | "authenticated" | "unauthenticated";

type AuthContextValue = {
  accessToken: string | null;
  status: SessionStatus;
  user: AuthUser | null;
  login: (input: LoginInput) => Promise<void>;
  register: (input: RegisterInput) => Promise<void>;
  refreshSession: () => Promise<string | null>;
  logout: () => Promise<void>;
  updateProfile: (name: string) => Promise<void>;
  uploadAvatar: (file: File) => Promise<AuthUser>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: Readonly<{ children: React.ReactNode }>) {
  const [accessToken, setAccessToken] = useState<string | null>(null);
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [user, setUser] = useState<AuthUser | null>(null);

  const applySession = useCallback((response: AuthResponse) => {
    setAccessToken(response.access_token);
    setUser(response.user);
    setStatus("authenticated");
  }, []);

  const clearSession = useCallback(() => {
    setAccessToken(null);
    setUser(null);
    setStatus("unauthenticated");
  }, []);

  const refreshSession = useCallback(async (): Promise<string | null> => {
    try {
      const response = await authAPI.refresh();
      applySession(response);
      return response.access_token;
    } catch {
      clearSession();
      return null;
    }
  }, [applySession, clearSession]);

  useEffect(() => {
    let isCurrent = true;

    async function restoreSession() {
      try {
        const response = await authAPI.refresh();
        if (isCurrent) {
          applySession(response);
        }
      } catch {
        if (isCurrent) {
          clearSession();
        }
      }
    }

    void restoreSession();
    return () => {
      isCurrent = false;
    };
  }, [applySession, clearSession]);

  const value = useMemo<AuthContextValue>(
    () => ({
      accessToken,
      status,
      user,
      login: async (input) => applySession(await authAPI.login(input)),
      register: async (input) => applySession(await authAPI.register(input)),
      refreshSession,
      logout: async () => {
        try {
          await authAPI.logout();
        } finally {
          clearSession();
        }
      },
      updateProfile: async (name) => {
        if (!accessToken) throw new Error("Нет активной сессии");
        setUser(await authAPI.updateProfile(accessToken, name));
      },
      uploadAvatar: async (file) => {
        if (!accessToken) throw new Error("Нет активной сессии");
        const updatedUser = await authAPI.uploadAvatar(accessToken, file);
        setUser(updatedUser);
        return updatedUser;
      },
    }),
    [accessToken, applySession, clearSession, refreshSession, status, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
