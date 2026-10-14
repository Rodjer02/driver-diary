export type Payment = "cash" | "card";

export type Trip = {
  id: string;
  start: string;
  end: string;
  amount: number;
  payment: Payment;
  commission: number;
};

export type PaymentTotals = { count: number; amount: number };

export type Summary = {
  trips: number;
  revenue: number;
  commission: number;
  net: number;
  cash: PaymentTotals;
  card: PaymentTotals;
};

export type DayResponse = { date: string; summary: Summary; trips: Trip[] };
export type DaysResponse = { days: string[]; tz: string };
export type NewTrip = Omit<Trip, "id">;

export type AddResult =
  | { kind: "created" | "duplicate"; trip: Trip }
  | { kind: "error"; message: string };

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(path, { cache: "no-store" });
  const body = await res.json();
  if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`);
  return body as T;
}

export const fetchDays = () => getJSON<DaysResponse>("/api/days");
export const fetchDay = (date: string) => getJSON<DayResponse>(`/api/day?date=${encodeURIComponent(date)}`);

export async function addTrip(trip: NewTrip): Promise<AddResult> {
  const res = await fetch("/api/trips", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(trip),
  });
  const body = await res.json();
  if (res.status === 201) return { kind: "created", trip: body.trip };
  if (res.status === 200) return { kind: "duplicate", trip: body.trip };
  const fields = body.errors ? Object.values(body.errors as Record<string, string>).join("; ") : "";
  return { kind: "error", message: fields || body.error || `HTTP ${res.status}` };
}
