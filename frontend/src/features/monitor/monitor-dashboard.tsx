"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { createMonitor, listMonitors, type Monitor } from "@/features/monitor/api";
import { useAuth } from "@/features/auth/auth-provider";

const units = { seconds: 1, minutes: 60, hours: 3600 } as const;
type Unit = keyof typeof units;

function formatInterval(seconds: number): string {
  if (seconds % 3600 === 0) return `Каждые ${seconds / 3600} ч.`;
  if (seconds % 60 === 0) return `Каждые ${seconds / 60} мин.`;
  return `Каждые ${seconds} сек.`;
}

export function MonitorDashboard() {
  const { accessToken, refreshSession } = useAuth();
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [url, setURL] = useState("");
  const [amount, setAmount] = useState("5");
  const [unit, setUnit] = useState<Unit>("seconds");
  const [error, setError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const isMounted = useRef(true);

  useEffect(() => () => {
    isMounted.current = false;
  }, []);

  useEffect(() => {
    if (!accessToken) return;
    const controller = new AbortController();
    void listMonitors(refreshSession, controller.signal)
      .then((result) => {
        if (isMounted.current) {
          setMonitors(result);
          setIsLoading(false);
        }
      })
      .catch((caught: unknown) => {
        if (caught instanceof DOMException && caught.name === "AbortError") return;
        if (isMounted.current) {
          setError("Не удалось загрузить сайты.");
          setIsLoading(false);
        }
      });
    return () => controller.abort();
  }, [accessToken, refreshSession]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!accessToken) return;
    const intervalSeconds = Number(amount) * units[unit];
    if (!Number.isInteger(intervalSeconds) || intervalSeconds <= 0) {
      setError("Укажите положительное целое число.");
      return;
    }
    setError("");
    setIsSaving(true);
    try {
      const monitor = await createMonitor(refreshSession, { url, interval_seconds: intervalSeconds });
      if (isMounted.current) {
        setMonitors((current) => [...current, monitor]);
        setURL("");
        setIsFormOpen(false);
      }
    } catch {
      if (isMounted.current) setError("Не удалось создать точку мониторинга.");
    } finally {
      if (isMounted.current) setIsSaving(false);
    }
  }

  return (
    <main className="mx-auto w-full max-w-6xl px-6 py-10">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div><h1 className="text-2xl font-semibold tracking-tight">Точки мониторинга</h1><p className="mt-1 text-sm text-zinc-500">Следите за доступностью ваших сайтов.</p></div>
        <button className="rounded-lg bg-blue-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-800" type="button" onClick={() => { setError(""); setIsFormOpen(true); }}>Добавить сайт</button>
      </div>

      {isFormOpen ? <form className="mt-8 rounded-xl border border-zinc-200 bg-white p-6 shadow-sm" onSubmit={handleSubmit}>
        <h2 className="text-lg font-semibold">Новый сайт</h2>
        <label className="mt-5 block text-sm font-medium" htmlFor="monitor-url">URL сайта</label>
        <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600" id="monitor-url" type="url" required placeholder="https://example.com" value={url} onChange={(event) => setURL(event.target.value)} />
        <div className="mt-4 grid gap-4 sm:grid-cols-2"><div><label className="block text-sm font-medium" htmlFor="monitor-amount">Частота</label><input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5" id="monitor-amount" type="number" min="1" step="1" required value={amount} onChange={(event) => setAmount(event.target.value)} /></div><div><label className="block text-sm font-medium" htmlFor="monitor-unit">Единица времени</label><select className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5" id="monitor-unit" value={unit} onChange={(event) => setUnit(event.target.value as Unit)}><option value="seconds">Секунды</option><option value="minutes">Минуты</option><option value="hours">Часы</option></select></div></div>
        {error ? <p className="mt-4 text-sm text-red-700" role="alert">{error}</p> : null}
        <div className="mt-6 flex gap-3"><button className="rounded-lg bg-blue-700 px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60" type="submit" disabled={isSaving}>{isSaving ? "Создаём…" : "Создать"}</button><button className="rounded-lg border border-zinc-300 px-4 py-2.5 text-sm font-medium" type="button" onClick={() => setIsFormOpen(false)}>Отмена</button></div>
      </form> : null}

      {error && !isFormOpen ? <p className="mt-6 text-sm text-red-700" role="alert">{error}</p> : null}
      <section className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">{monitors.map((monitor) => <article className="rounded-xl border border-zinc-200 bg-white p-5 shadow-sm" key={monitor.id}><p className="truncate font-medium" title={monitor.url}>{monitor.url}</p><p className="mt-3 text-sm text-zinc-500">{formatInterval(monitor.interval_seconds)}</p></article>)}</section>
      {isLoading ? <p className="mt-12 text-center text-sm text-zinc-500">Загружаем сайты…</p> : null}
      {!isLoading && monitors.length === 0 && !isFormOpen ? <p className="mt-12 text-center text-sm text-zinc-500">Пока нет сайтов. Добавьте первый сайт для мониторинга.</p> : null}
    </main>
  );
}
