"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";
import { fetchDays } from "@/lib/api";
import { dayLabel, todayIn } from "@/lib/format";
import TripForm from "./TripForm";
import { Close } from "./icons";

export default function NewTripScreen() {
  const params = useSearchParams();
  // пояс уточняется у API; до ответа — пояс по умолчанию, чтобы форма не ждала сеть
  const [tz, setTz] = useState("+05:00");

  useEffect(() => {
    fetchDays()
      .then((r) => setTz(r.tz))
      .catch(() => {});
  }, []);

  const raw = params.get("date");
  const date = raw && /^\d{4}-\d{2}-\d{2}$/.test(raw) ? raw : todayIn(tz);

  return (
    <div className="page">
      <div className="screen">
        <div className="screen-bar">
          <Link href={`/?date=${date}`} className="icon-btn" aria-label="Закрыть и вернуться к смене">
            <Close />
          </Link>
          <span className="muted">{dayLabel(date)}</span>
          <span className="icon-spacer" />
        </div>
        <h1 className="screen-title">Новая поездка</h1>
        <TripForm date={date} tz={tz} variant="screen" />
      </div>
    </div>
  );
}
