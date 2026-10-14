"use client";

import { useEffect, useMemo, useRef } from "react";
import { addDays, dayParts, monthLabel } from "@/lib/format";
import { Calendar, ChevronLeft, ChevronRight } from "./icons";

const STRIP_DAYS = 30;

type Props = {
  date: string;
  today: string;
  days: string[]; // дни, в которые есть поездки
  onPick: (day: string) => void;
};

/** Диапазон ленты: 30 дней, заканчивается сегодня; если выбран далёкий день — вокруг него. */
function stripRange(date: string, today: string): string[] {
  let end = date > today ? date : today;
  if (addDays(date, STRIP_DAYS - 1) < end) end = addDays(date, 7);
  return Array.from({ length: STRIP_DAYS }, (_, i) => addDays(end, i - STRIP_DAYS + 1));
}

export default function DayNav({ date, today, days, onPick }: Props) {
  const withTrips = useMemo(() => new Set(days), [days]);
  const prev = [...days].reverse().find((d) => d < date);
  const next = days.find((d) => d > date);
  const strip = stripRange(date, today);

  const stripRef = useRef<HTMLDivElement>(null);
  const pickerRef = useRef<HTMLInputElement>(null);

  // первый показ — сразу на выбранный день, дальше — плавная прокрутка ленты
  const mounted = useRef(false);
  useEffect(() => {
    const box = stripRef.current;
    const el = box?.querySelector<HTMLElement>('[aria-pressed="true"]');
    if (!box || !el) return;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const left = el.offsetLeft - (box.clientWidth - el.offsetWidth) / 2;
    box.scrollTo({ left, behavior: mounted.current && !reduce ? "smooth" : "auto" });
    mounted.current = true;
  }, [date]);

  function openPicker() {
    const el = pickerRef.current;
    if (!el) return;
    try {
      el.showPicker();
    } catch {
      el.focus();
    }
  }

  return (
    <>
      <header className="topbar">
        <span className="brand">Дневник смен</span>
        <div className="topbar-nav">
          <span className="month">{monthLabel(date)}</span>
          <div className="icon-row">
            <button type="button" className="icon-btn" onClick={() => prev && onPick(prev)} disabled={!prev} aria-label="Предыдущий день с поездками">
              <ChevronLeft />
            </button>
            <button type="button" className="icon-btn" onClick={() => next && onPick(next)} disabled={!next} aria-label="Следующий день с поездками">
              <ChevronRight />
            </button>
            <span className="picker-wrap">
              <button type="button" className="icon-btn" onClick={openPicker} aria-label="Открыть календарь">
                <Calendar />
              </button>
              <input
                ref={pickerRef}
                type="date"
                className="picker-input"
                value={date}
                onChange={(e) => e.target.value && onPick(e.target.value)}
                tabIndex={-1}
                aria-hidden
              />
            </span>
          </div>
        </div>
      </header>

      <div className="strip" ref={stripRef}>
        {strip.map((d) => {
          const p = dayParts(d);
          const selected = d === date;
          const has = withTrips.has(d);
          return (
            <button
              key={d}
              type="button"
              className={`day${selected ? " selected" : ""}${d === today ? " today" : ""}`}
              aria-pressed={selected}
              aria-label={p.aria + (has ? ", есть поездки" : "")}
              onClick={() => !selected && onPick(d)}
            >
              <span className="day-wd">{p.wd}</span>
              <span className="day-num">{p.num}</span>
              <span className={`day-dot${has ? " on" : ""}`} />
            </button>
          );
        })}
      </div>
    </>
  );
}
