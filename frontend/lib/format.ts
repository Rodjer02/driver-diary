const WD = ["Вс", "Пн", "Вт", "Ср", "Чт", "Пт", "Сб"];
const MON_GEN = ["января", "февраля", "марта", "апреля", "мая", "июня", "июля", "августа", "сентября", "октября", "ноября", "декабря"];
const MON_NOM = ["Январь", "Февраль", "Март", "Апрель", "Май", "Июнь", "Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь"];

/** 14280 -> "14 280" */
export const num = (n: number) => String(n).replace(/\B(?=(\d{3})+(?!\d))/g, " ");

const utc = (day: string) => new Date(`${day}T00:00:00Z`);

export function addDays(day: string, n: number): string {
  const d = utc(day);
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
}

/** "2026-10-01" -> "Чт, 1 октября" */
export function dayLabel(day: string): string {
  const d = utc(day);
  return `${WD[d.getUTCDay()]}, ${d.getUTCDate()} ${MON_GEN[d.getUTCMonth()]}`;
}

/** "2026-10-01" -> "Октябрь 2026" */
export function monthLabel(day: string): string {
  const d = utc(day);
  return `${MON_NOM[d.getUTCMonth()]} ${d.getUTCFullYear()}`;
}

export function dayParts(day: string) {
  const d = utc(day);
  return { wd: WD[d.getUTCDay()], num: d.getUTCDate(), aria: dayLabel(day) };
}

/** Сегодня в поясе водителя ("+05:00"). */
export function todayIn(tz: string): string {
  const m = /^([+-])(\d{2}):(\d{2})$/.exec(tz);
  const offsetMin = m ? (m[1] === "-" ? -1 : 1) * (Number(m[2]) * 60 + Number(m[3])) : 0;
  return new Date(Date.now() + offsetMin * 60_000).toISOString().slice(0, 10);
}

export function tripsWord(n: number): string {
  const m10 = n % 10;
  const m100 = n % 100;
  if (m10 === 1 && m100 !== 11) return "поездка";
  if (m10 >= 2 && m10 <= 4 && (m100 < 10 || m100 >= 20)) return "поездки";
  return "поездок";
}

/** Время "ЧЧ:ММ" из ISO в поясе водителя (сервер отдаёт время уже в нём). */
export const hhmm = (iso: string) => iso.slice(11, 16);

/** Поездка закончилась в другие сутки. */
export const isNight = (start: string, end: string) => start.slice(0, 10) !== end.slice(0, 10);

/** Маска ввода времени: оставляет цифры и ставит двоеточие: "2350" -> "23:50". */
export function maskTime(v: string): string {
  const d = v.replace(/\D/g, "").slice(0, 4);
  return d.length > 2 ? `${d.slice(0, 2)}:${d.slice(2)}` : d;
}

export const isTime = (t: string) => /^([01]\d|2[0-3]):[0-5]\d$/.test(t);

export const digits = (v: string) => v.replace(/\D/g, "");
