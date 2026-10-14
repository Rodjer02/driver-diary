"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDay, fetchDays, type DayResponse, type Summary, type Trip } from "@/lib/api";
import { dayLabel, hhmm, isNight, num, todayIn, tripsWord } from "@/lib/format";
import { useTween } from "@/lib/useTween";
import DayNav from "./DayNav";
import TripForm from "./TripForm";
import { Plus } from "./icons";

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

export default function Dashboard() {
  const router = useRouter();
  const params = useSearchParams();
  const urlDate = params.get("date");

  const [days, setDays] = useState<string[] | null>(null);
  const [tz, setTz] = useState("+05:00");
  const [data, setData] = useState<DayResponse | null>(null);
  const [error, setError] = useState("");

  const loadDays = useCallback(
    () =>
      fetchDays().then((r) => {
        setDays(r.days);
        setTz(r.tz);
        return r;
      }),
    [],
  );

  useEffect(() => {
    loadDays().catch((e: Error) => setError(e.message));
  }, [loadDays]);

  const today = todayIn(tz);
  // без ?date= открываем последний день с поездками
  const date = urlDate && DATE_RE.test(urlDate) ? urlDate : days ? (days.at(-1) ?? today) : null;

  // при быстром переключении дней применяется только ответ на последний запрос
  const latest = useRef("");
  const loadDay = useCallback((d: string) => {
    latest.current = d;
    return fetchDay(d)
      .then((r) => {
        if (latest.current !== d) return;
        setData(r);
        setError("");
      })
      .catch((e: Error) => latest.current === d && setError(e.message));
  }, []);

  useEffect(() => {
    if (date) loadDay(date);
  }, [date, loadDay]);

  const pick = (d: string) => router.replace(`/?date=${d}`, { scroll: false });

  const onSaved = useCallback(() => {
    if (!date) return;
    loadDays();
    loadDay(date);
  }, [date, loadDays, loadDay]);

  if (!date) {
    return <div className="page">{error ? <ErrorBox text={error} /> : <div className="loading">Загрузка…</div>}</div>;
  }

  // пока грузится новый день, на экране остаётся прошлый — приглушённый, без «провала»
  const shown = data;
  const loading = !data || data.date !== date;

  return (
    <div className="page">
      <div className="container">
        <DayNav date={date} today={today} days={days ?? []} onPick={pick} />

        <div className="layout">
          <main className="main">
            {error && <ErrorBox text={error} />}
            <div className={`dimmable${loading && shown ? " is-loading" : ""}`} aria-busy={loading}>
              <Hero date={shown?.date ?? date} summary={shown?.summary ?? null} />
              {shown && shown.trips.length > 0 && <Split s={shown.summary} />}
              {shown && (
                <div key={shown.date} className="day-body fade-in">
                  {shown.trips.length > 0 ? (
                    <>
                      <TripList trips={shown.trips} />
                      <TripTable trips={shown.trips} s={shown.summary} />
                    </>
                  ) : (
                    <div className="empty">
                      <span className="empty-title">В этот день поездок нет</span>
                      <span className="muted">Добавьте поездку — итоги посчитаются сразу</span>
                    </div>
                  )}
                </div>
              )}
            </div>
            <Link href={`/new?date=${date}`} className="cta add-cta">
              <Plus />
              Добавить поездку
            </Link>
          </main>

          <aside className="aside">
            <div className="aside-head">
              <span className="aside-title">Новая поездка</span>
              <span className="muted small">в день: {dayLabel(date)}</span>
            </div>
            <TripForm key={date} date={date} tz={tz} variant="aside" onSaved={onSaved} />
          </aside>
        </div>
      </div>
    </div>
  );
}

function ErrorBox({ text }: { text: string }) {
  return (
    <div role="alert" className="error-box">
      Не удалось загрузить данные: {text}
    </div>
  );
}

function Hero({ date, summary: s }: { date: string; summary: Summary | null }) {
  const empty = !s || s.trips === 0;
  // первое значение — сразу, анимация только при смене дня или после добавления поездки
  const seen = useRef(false);
  const net = useTween(s?.net ?? 0, seen.current ? 420 : 0);
  useEffect(() => {
    if (s) seen.current = true;
  }, [s]);
  return (
    <section className="hero">
      <div className="hero-main">
        <span className="muted">На руки · {dayLabel(date)}</span>
        <div className={`hero-net${empty ? " zero" : ""}`}>
          <span className="hero-value">{s ? num(net) : "—"}</span>
          <span className="hero-cur">₸</span>
        </div>
        {s && s.trips > 0 && (
          <span className="hero-line muted">
            {num(s.revenue)} выручка · −{num(s.commission)} комиссия · {s.trips} {tripsWord(s.trips)}
          </span>
        )}
      </div>
      {s && s.trips > 0 && (
        <dl className="hero-stats">
          <dt>Поездок</dt>
          <dt>Выручка</dt>
          <dt>Комиссия</dt>
          <dd>{s.trips}</dd>
          <dd>{num(s.revenue)} ₸</dd>
          <dd>−{num(s.commission)} ₸</dd>
        </dl>
      )}
    </section>
  );
}

function Split({ s }: { s: Summary }) {
  const pct = s.revenue ? Math.round((s.cash.amount / s.revenue) * 1000) / 10 : 0;
  const aria = `Наличные ${Math.round(pct)} процентов, карта ${100 - Math.round(pct)} процентов`;
  return (
    <section className="split">
      <div className={`split-bar${s.cash.amount && s.card.amount ? "" : " solo"}`} role="img" aria-label={aria}>
        <div className="seg cash" style={{ width: `${pct}%` }} />
        <div className="seg card" />
      </div>

      <div className="split-cols">
        <div className="split-col">
          <span className="split-label">
            <span className="dot cash" />
            Наличные · {s.cash.count}
          </span>
          <span className="split-sum">{num(s.cash.amount)} ₸</span>
          <span className="muted small">уже в кармане</span>
        </div>
        <div className="split-col">
          <span className="split-label">
            <span className="dot card" />
            Карта · {s.card.count}
          </span>
          <span className="split-sum">{num(s.card.amount)} ₸</span>
          <span className="muted small">придут позже</span>
        </div>
      </div>

      <div className="split-line">
        <span>
          <span className="dot cash" />
          Наличные · {s.cash.count} <b>{num(s.cash.amount)} ₸</b> <span className="muted">уже в кармане</span>
        </span>
        <span>
          <span className="muted">придут позже</span> <b>{num(s.card.amount)} ₸</b> Карта · {s.card.count}
          <span className="dot card" />
        </span>
      </div>
    </section>
  );
}

const payLabel = (t: Trip) => (t.payment === "cash" ? "Наличные" : "Карта");

function TimeCell({ t }: { t: Trip }) {
  return (
    <>
      {hhmm(t.start)}–{hhmm(t.end)}
      {isNight(t.start, t.end) && <span className="night">ночная</span>}
    </>
  );
}

function TripList({ trips }: { trips: Trip[] }) {
  return (
    <section className="trip-list">
      <div className="caps-row">
        <span>Поездки</span>
        <span>На руки</span>
      </div>
      {trips.map((t) => (
        <div key={t.id} className="trip-row">
          <div className="trip-left">
            <span className="trip-time">
              <TimeCell t={t} />
            </span>
            <span className="trip-pay">
              <span className={`dot ${t.payment}`} />
              {payLabel(t)}
            </span>
          </div>
          <div className="trip-right">
            <span className="trip-net">{num(t.amount - t.commission)} ₸</span>
            <span className="muted small">из {num(t.amount)}</span>
          </div>
        </div>
      ))}
    </section>
  );
}

function TripTable({ trips, s }: { trips: Trip[]; s: Summary }) {
  return (
    <section className="trip-table">
      <table>
        <thead>
          <tr>
            <th>Время</th>
            <th>Оплата</th>
            <th className="r">Сумма</th>
            <th className="r">Комиссия</th>
            <th className="r">На руки</th>
          </tr>
        </thead>
        <tbody>
          {trips.map((t) => (
            <tr key={t.id}>
              <td className="strong">
                <TimeCell t={t} />
              </td>
              <td className="muted">
                <span className="trip-pay">
                  <span className={`dot ${t.payment}`} />
                  {payLabel(t)}
                </span>
              </td>
              <td className="r">{num(t.amount)}</td>
              <td className="r muted">−{num(t.commission)}</td>
              <td className="r strong">{num(t.amount - t.commission)} ₸</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={2}>Итого</td>
            <td className="r">{num(s.revenue)}</td>
            <td className="r muted">−{num(s.commission)}</td>
            <td className="r bold">{num(s.net)} ₸</td>
          </tr>
        </tfoot>
      </table>
    </section>
  );
}
