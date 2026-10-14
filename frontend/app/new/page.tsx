import type { Metadata } from "next";
import { Suspense } from "react";
import NewTripScreen from "@/components/NewTripScreen";

export const metadata: Metadata = { title: "Новая поездка · Дневник смен" };

export default function Page() {
  return (
    <Suspense fallback={<div className="page"><div className="loading">Загрузка…</div></div>}>
      <NewTripScreen />
    </Suspense>
  );
}
