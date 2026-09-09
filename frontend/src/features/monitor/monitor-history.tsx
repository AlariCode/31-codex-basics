"use client";

import { useEffect, useState } from "react";
import type { Monitor, MonitorStats, StatsPoint } from "./api";

const reasons: Record<string, string> = {
  timeout: "Превышено время ожидания", dns: "Ошибка DNS", tls: "Ошибка TLS",
  network: "Ошибка соединения", forbidden_address: "Адрес запрещён",
};

export function MonitorStatus({ monitor }: Readonly<{ monitor: Monitor }>) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  const checked = monitor.last_checked_at ? new Date(monitor.last_checked_at) : null;
  const stale = checked && now - checked.getTime() > (monitor.interval_seconds + 15) * 1000;
  let label = "Ожидаем проверку";
  let color = "bg-zinc-100 text-zinc-600";
  if (monitor.last_status === "blocked") {
    label = "Адрес запрещён";
    color = "bg-amber-50 text-amber-800";
  } else if (stale) {
    label = "Данные устарели";
    color = "bg-amber-50 text-amber-800";
  } else if (monitor.last_status === "up") {
    label = "Работает";
    color = "bg-emerald-50 text-emerald-800";
  } else if (monitor.last_status === "down") {
    label = "Недоступен";
    color = "bg-red-50 text-red-700";
  }
  return <div className="mt-4 space-y-2">
    <span className={`inline-flex rounded-full px-2.5 py-1 text-xs font-medium ${color}`}>{label}</span>
    <p className="min-h-8 text-xs leading-4 text-zinc-500">
      {checked ? <>Проверка: <time dateTime={monitor.last_checked_at ?? undefined}>{checked.toLocaleString("ru-RU")}</time>
        {monitor.last_http_status ? ` · HTTP ${monitor.last_http_status}` : ""}
        {monitor.last_error ? ` · ${reasons[monitor.last_error] ?? "Ошибка проверки"}` : ""}</> : "Первая проверка запускается автоматически."}
    </p>
  </div>;
}

function pointDescription(point: StatsPoint): string {
  const start = new Date(point.start).toLocaleString("ru-RU");
  const end = new Date(point.end).toLocaleString("ru-RU");
  if (point.uptime_percent === null) return `${start} — ${end}: нет данных`;
  return `${start} — ${end}: ${point.uptime_percent.toFixed(1)}%; успешно ${point.successes}, неуспешно ${point.failures}`;
}

export function MonitorHistory({ stats, loading = false }: Readonly<{ stats?: MonitorStats; loading?: boolean }>) {
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const points = stats?.points ?? [];
  const selected = activeIndex === null ? null : points[activeIndex];
  return <div className="mt-4 border-t border-zinc-100 pt-4">
    <div className="flex items-baseline justify-between gap-2 text-xs text-zinc-500">
      <span>Успешные проверки</span>
      <strong className="text-base font-semibold text-zinc-900">{stats?.uptime_percent != null ? `${stats.uptime_percent.toFixed(1)}%` : "Нет данных"}</strong>
    </div>
    <div className="relative mt-3">
      <div className="flex h-16 items-end gap-px" role="group" aria-label="График доступности">
        {points.map((point, index) => <button
          key={point.start}
          type="button"
          className="group flex h-full min-w-0 flex-1 items-end rounded-sm bg-zinc-100 outline-offset-2 focus-visible:outline-2 focus-visible:outline-blue-700"
          aria-label={pointDescription(point)}
          tabIndex={index === (activeIndex ?? 0) ? 0 : -1}
          onMouseEnter={() => setActiveIndex(index)} onMouseLeave={() => setActiveIndex(null)}
          onFocus={() => setActiveIndex(index)} onBlur={() => setActiveIndex(null)}
          onKeyDown={(event) => {
            const offset = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
            if (!offset) return;
            event.preventDefault();
            const target = offset > 0 ? event.currentTarget.nextElementSibling : event.currentTarget.previousElementSibling;
            if (target instanceof HTMLButtonElement) target.focus();
          }}
        ><span className={`w-full rounded-sm ${point.uptime_percent === null ? "bg-zinc-200" : point.failures > 0 ? "bg-red-500" : "bg-emerald-500"}`}
          style={{ height: point.uptime_percent === null ? "100%" : `${Math.max(5, point.uptime_percent)}%` }} /></button>)}
      </div>
      {selected ? <div role="tooltip" className="pointer-events-none absolute bottom-full left-0 z-10 mb-2 w-full rounded-md bg-zinc-900 px-3 py-2 text-xs text-white shadow-lg">{pointDescription(selected)}</div> : null}
    </div>
    <div className="mt-2 flex justify-between gap-2 text-[10px] text-zinc-500">
      <span>{points[0] ? new Date(points[0].start).toLocaleString("ru-RU", { day: "numeric", month: "short", hour: "2-digit", minute: "2-digit" }) : loading ? "Загружаем историю…" : "Нет данных за период"}</span>
      <span>{points.at(-1) ? new Date(points.at(-1)!.end).toLocaleTimeString("ru-RU", { hour: "2-digit", minute: "2-digit" }) : ""}</span>
    </div>
  </div>;
}
