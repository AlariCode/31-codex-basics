"use client";

import { useEffect, useRef, useState } from "react";
import type { FormEvent } from "react";

import { isRequestCanceled } from "@/features/api-client";
import { createMonitor, deleteMonitor, faviconURL, listMonitors, updateMonitor, type Monitor } from "@/features/monitor/api";
import { useAuth } from "@/features/auth/auth-provider";

const units = { seconds: 1, minutes: 60, hours: 3600 } as const;
type Unit = keyof typeof units;

function formatInterval(seconds: number): string {
  if (seconds % 3600 === 0) return `Каждые ${seconds / 3600} ч.`;
  if (seconds % 60 === 0) return `Каждые ${seconds / 60} мин.`;
  return `Каждые ${seconds} сек.`;
}

function intervalFormValue(seconds: number): { amount: string; unit: Unit } {
  if (seconds % units.hours === 0) return { amount: String(seconds / units.hours), unit: "hours" };
  if (seconds % units.minutes === 0) return { amount: String(seconds / units.minutes), unit: "minutes" };
  return { amount: String(seconds), unit: "seconds" };
}

function DefaultSiteIcon() {
  return (
    <span className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-blue-100 text-blue-700" data-testid="default-site-icon" aria-hidden="true">
      <svg className="size-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" focusable="false">
        <circle cx="12" cy="12" r="9" />
        <path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" />
      </svg>
    </span>
  );
}

function MonitorIcon({ source, url }: Readonly<{ source: string; url: string }>) {
  const [failedSource, setFailedSource] = useState<string | null>(null);
  const faviconSource = faviconURL(source);
  const imageSource = faviconSource ? `${faviconSource}?site=${encodeURIComponent(url)}` : null;

  if (!imageSource || failedSource === imageSource) {
    return <DefaultSiteIcon />;
  }

  // The API image URL is resolved at runtime; a native image lets this card switch to its fallback on a failed request.
  // eslint-disable-next-line @next/next/no-img-element
  return <img className="size-10 shrink-0 rounded-lg border border-zinc-200 bg-white object-contain p-1" data-testid="monitor-favicon" src={imageSource} alt="" onError={() => setFailedSource(imageSource)} />;
}

export function MonitorDashboard() {
  const { accessToken, refreshSession } = useAuth();
  const [monitors, setMonitors] = useState<Monitor[]>([]);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [editingMonitorID, setEditingMonitorID] = useState<string | null>(null);
  const [deletingMonitor, setDeletingMonitor] = useState<Monitor | null>(null);
  const [url, setURL] = useState("");
  const [amount, setAmount] = useState("5");
  const [unit, setUnit] = useState<Unit>("seconds");
  const [error, setError] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const isMounted = useRef(false);

  useEffect(() => {
    isMounted.current = true;
    return () => {
      isMounted.current = false;
    };
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
        if (isRequestCanceled(caught)) return;
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
      const monitor = editingMonitorID
        ? await updateMonitor(refreshSession, editingMonitorID, { url, interval_seconds: intervalSeconds })
        : await createMonitor(refreshSession, { url, interval_seconds: intervalSeconds });
      if (isMounted.current) {
        setMonitors((current) => editingMonitorID
          ? current.map((value) => value.id === monitor.id ? monitor : value)
          : [...current, monitor]);
        setURL("");
        setEditingMonitorID(null);
        setIsFormOpen(false);
      }
    } catch {
      if (isMounted.current) setError(editingMonitorID ? "Не удалось обновить сайт." : "Не удалось создать точку мониторинга.");
    } finally {
      if (isMounted.current) setIsSaving(false);
    }
  }

  function openCreateForm() {
    setError("");
    setEditingMonitorID(null);
    setURL("");
    setAmount("5");
    setUnit("seconds");
    setIsFormOpen(true);
  }

  function openEditForm(monitor: Monitor) {
    const interval = intervalFormValue(monitor.interval_seconds);
    setError("");
    setEditingMonitorID(monitor.id);
    setURL(monitor.url);
    setAmount(interval.amount);
    setUnit(interval.unit);
    setIsFormOpen(true);
  }

  async function handleDelete() {
    if (!deletingMonitor) return;
    setIsDeleting(true);
    setError("");
    try {
      await deleteMonitor(refreshSession, deletingMonitor.id);
      if (isMounted.current) {
        setMonitors((current) => current.filter((value) => value.id !== deletingMonitor.id));
        setDeletingMonitor(null);
      }
    } catch {
      if (isMounted.current) setError("Не удалось удалить сайт.");
    } finally {
      if (isMounted.current) setIsDeleting(false);
    }
  }

  return (
    <main className="mx-auto w-full max-w-6xl px-6 py-10">
      <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-center">
        <div><h1 className="text-2xl font-semibold tracking-tight">Точки мониторинга</h1><p className="mt-1 text-sm text-zinc-500">Следите за доступностью ваших сайтов.</p></div>
        <button className="rounded-lg bg-blue-700 px-4 py-2.5 text-sm font-medium text-white hover:bg-blue-800" type="button" onClick={openCreateForm}>Добавить сайт</button>
      </div>

      {isFormOpen ? <form className="mt-8 rounded-xl border border-zinc-200 bg-white p-6 shadow-sm" onSubmit={handleSubmit}>
        <h2 className="text-lg font-semibold">{editingMonitorID ? "Редактировать сайт" : "Новый сайт"}</h2>
        <label className="mt-5 block text-sm font-medium" htmlFor="monitor-url">URL сайта</label>
        <input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5 outline-none focus:border-blue-600" id="monitor-url" type="url" required placeholder="https://example.com" value={url} onChange={(event) => setURL(event.target.value)} />
        <div className="mt-4 grid gap-4 sm:grid-cols-2"><div><label className="block text-sm font-medium" htmlFor="monitor-amount">Частота</label><input className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5" id="monitor-amount" type="number" min="1" step="1" required value={amount} onChange={(event) => setAmount(event.target.value)} /></div><div><label className="block text-sm font-medium" htmlFor="monitor-unit">Единица времени</label><select className="mt-2 w-full rounded-lg border border-zinc-300 px-3 py-2.5" id="monitor-unit" value={unit} onChange={(event) => setUnit(event.target.value as Unit)}><option value="seconds">Секунды</option><option value="minutes">Минуты</option><option value="hours">Часы</option></select></div></div>
        {error ? <p className="mt-4 text-sm text-red-700" role="alert">{error}</p> : null}
        <div className="mt-6 flex gap-3"><button className="rounded-lg bg-blue-700 px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60" type="submit" disabled={isSaving}>{isSaving ? "Сохраняем…" : editingMonitorID ? "Сохранить" : "Создать"}</button><button className="rounded-lg border border-zinc-300 px-4 py-2.5 text-sm font-medium" type="button" onClick={() => { setIsFormOpen(false); setEditingMonitorID(null); }}>Отмена</button></div>
      </form> : null}

      {error && !isFormOpen && !deletingMonitor ? <p className="mt-6 text-sm text-red-700" role="alert">{error}</p> : null}
      <section className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">{monitors.map((monitor) => <article className="rounded-xl border border-zinc-200 bg-white p-5 shadow-sm" key={monitor.id}><div className="flex items-start gap-3"><MonitorIcon source={monitor.favicon_url} url={monitor.url} /><p className="min-w-0 truncate pt-2 font-medium" title={monitor.url}>{monitor.url}</p></div><p className="mt-3 text-sm text-zinc-500">{formatInterval(monitor.interval_seconds)}</p><div className="mt-5 flex gap-2"><button className="rounded-lg border border-zinc-300 px-3 py-2 text-sm font-medium hover:bg-zinc-50" type="button" onClick={() => openEditForm(monitor)}>Редактировать</button><button className="rounded-lg border border-red-200 px-3 py-2 text-sm font-medium text-red-700 hover:bg-red-50" type="button" onClick={() => { setError(""); setDeletingMonitor(monitor); }}>Удалить</button></div></article>)}</section>
      {isLoading ? <p className="mt-12 text-center text-sm text-zinc-500">Загружаем сайты…</p> : null}
      {!isLoading && monitors.length === 0 && !isFormOpen ? <p className="mt-12 text-center text-sm text-zinc-500">Пока нет сайтов. Добавьте первый сайт для мониторинга.</p> : null}
      {deletingMonitor ? <div className="fixed inset-0 z-10 flex items-center justify-center bg-black/40 px-6" role="presentation"><div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl" role="dialog" aria-modal="true" aria-labelledby="delete-monitor-title"><h2 className="text-lg font-semibold" id="delete-monitor-title">Удалить сайт?</h2><p className="mt-3 text-sm text-zinc-600">Сайт <span className="font-medium text-zinc-900">{deletingMonitor.url}</span> будет удалён без возможности восстановления.</p>{error ? <p className="mt-4 text-sm text-red-700" role="alert">{error}</p> : null}<div className="mt-6 flex justify-end gap-3"><button className="rounded-lg border border-zinc-300 px-4 py-2.5 text-sm font-medium" type="button" disabled={isDeleting} onClick={() => setDeletingMonitor(null)}>Отмена</button><button className="rounded-lg bg-red-700 px-4 py-2.5 text-sm font-medium text-white disabled:opacity-60" type="button" disabled={isDeleting} onClick={() => void handleDelete()}>{isDeleting ? "Удаляем…" : "Удалить"}</button></div></div></div> : null}
    </main>
  );
}
