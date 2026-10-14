"use client";

import { useState } from "react";
import { addTrip, type Payment } from "@/lib/api";
import { addDays, digits, isTime, maskTime, num } from "@/lib/format";
import Feedback, { type FeedbackState } from "./Feedback";
import { Moon } from "./icons";

type Props = {
  date: string;
  tz: string;
  /** screen — отдельный экран на телефоне, aside — колонка на десктопе */
  variant: "screen" | "aside";
  onSaved?: () => void;
};

const empty = { start: "", end: "", amount: "", commission: "" };

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

export default function TripForm({ date, tz, variant, onSaved }: Props) {
  const [f, setF] = useState(empty);
  const [pay, setPay] = useState<Payment>("cash");
  const [feedback, setFeedback] = useState<FeedbackState>(null);
  const [saving, setSaving] = useState(false);

  const amount = Number(f.amount) || 0;
  const commission = Number(f.commission) || 0;
  const night = isTime(f.start) && isTime(f.end) && f.end <= f.start;

  const set = (key: keyof typeof empty, transform: (v: string) => string) => (e: React.ChangeEvent<HTMLInputElement>) => {
    setF((prev) => ({ ...prev, [key]: transform(e.target.value) }));
    setFeedback(null);
  };

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const errs: string[] = [];
    if (!isTime(f.start)) errs.push("Укажите время начала в формате ЧЧ:ММ");
    if (!isTime(f.end)) errs.push("Укажите время окончания в формате ЧЧ:ММ");
    if (!(amount > 0)) errs.push("Сумма должна быть больше 0");
    if (f.commission === "" || commission > amount) errs.push("Комиссия должна быть от 0 до суммы");
    if (errs.length) {
      setFeedback({ kind: "errors", list: errs });
      return;
    }

    setSaving(true);
    const res = await addTrip({
      start: `${date}T${f.start}:00${tz}`,
      end: `${night ? addDays(date, 1) : date}T${f.end}:00${tz}`,
      amount,
      commission,
      payment: pay,
    }).catch(() => ({ kind: "error" as const, message: "Нет связи с сервером. Попробуйте ещё раз" }));
    setSaving(false);

    if (res.kind === "error") {
      setFeedback({ kind: "errors", list: res.message.split("; ").map(capitalize) });
      return;
    }
    setFeedback({ kind: res.kind });
    if (res.kind === "created") setF(empty);
    onSaved?.();
  }

  const big = variant === "screen";

  return (
    <form className={`trip-form trip-form-${variant}`} onSubmit={submit} noValidate>
      <div className="field-group">
        <div className="grid2">
          <label className="field">
            Начало
            <input className={big ? "big" : ""} inputMode="numeric" maxLength={5} placeholder="00:00" value={f.start} onChange={set("start", maskTime)} />
          </label>
          <label className="field">
            Окончание
            <input className={big ? "big" : ""} inputMode="numeric" maxLength={5} placeholder="00:00" value={f.end} onChange={set("end", maskTime)} />
          </label>
          {!big && (
            <>
              <label className="field">
                Сумма, ₸
                <input inputMode="numeric" placeholder="0" value={f.amount} onChange={set("amount", digits)} />
              </label>
              <label className="field">
                Комиссия, ₸
                <input inputMode="numeric" placeholder="0" value={f.commission} onChange={set("commission", digits)} />
              </label>
            </>
          )}
        </div>
        {night && (
          <span className="night-hint">
            <Moon />
            Ночная — закончится на следующие сутки
          </span>
        )}
      </div>

      {big && (
        <div className="grid2">
          <label className="field">
            Сумма, ₸
            <input className="big" inputMode="numeric" placeholder="0" value={f.amount} onChange={set("amount", digits)} />
          </label>
          <label className="field">
            Комиссия, ₸
            <input className="big" inputMode="numeric" placeholder="0" value={f.commission} onChange={set("commission", digits)} />
          </label>
        </div>
      )}

      <div className="field-group">
        {big && (
          <span className="field-label" id={`pay-${variant}`}>
            Оплата
          </span>
        )}
        <div className="grid2 tight" role="group" aria-label={big ? undefined : "Оплата"} aria-labelledby={big ? `pay-${variant}` : undefined}>
          {(["cash", "card"] as const).map((p) => (
            <button key={p} type="button" className={`pill${pay === p ? " active" : ""}`} aria-pressed={pay === p} onClick={() => setPay(p)}>
              <span className={`dot ${p}`} />
              {p === "cash" ? "Наличные" : "Карта"}
            </button>
          ))}
        </div>
      </div>

      {big && (
        <div className="net-preview">
          <span>На руки</span>
          <span className="net-preview-value">
            {num(Math.max(amount - commission, 0))} <span className="muted">₸</span>
          </span>
        </div>
      )}

      <div className="form-actions">
        <Feedback state={feedback} />
        <button type="submit" className="cta" disabled={saving}>
          {saving ? "Сохраняю…" : "Сохранить"}
        </button>
        {!big && <span className="form-note">Если окончание раньше начала, поездка ночная и закончится на следующие сутки.</span>}
      </div>
    </form>
  );
}
