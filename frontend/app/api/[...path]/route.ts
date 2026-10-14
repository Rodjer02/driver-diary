// Прокси /api/* -> Go API. Браузер ходит на тот же origin, поэтому CORS не нужен,
// а адрес бэкенда берётся из BACKEND_URL при каждом запросе, а не при сборке.

const backend = () => (process.env.BACKEND_URL ?? "http://localhost:8080").replace(/\/$/, "");

async function proxy(req: Request): Promise<Response> {
  const url = new URL(req.url);
  const target = `${backend()}${url.pathname}${url.search}`;
  const init: RequestInit = {
    method: req.method,
    headers: { "Content-Type": req.headers.get("content-type") ?? "application/json" },
    cache: "no-store",
  };
  if (req.method !== "GET" && req.method !== "HEAD") {
    init.body = await req.text();
  }

  try {
    const res = await fetch(target, init);
    return new Response(await res.text(), {
      status: res.status,
      headers: { "Content-Type": res.headers.get("content-type") ?? "application/json" },
    });
  } catch {
    return Response.json({ error: "API недоступен" }, { status: 502 });
  }
}

export const GET = proxy;
export const POST = proxy;
