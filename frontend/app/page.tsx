import { Suspense } from "react";
import Dashboard from "@/components/Dashboard";

export default function Page() {
  return (
    <Suspense fallback={<div className="page"><div className="loading">Загрузка…</div></div>}>
      <Dashboard />
    </Suspense>
  );
}
