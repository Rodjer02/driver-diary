package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"driverdiary/internal/api"
	"driverdiary/internal/store"
	"driverdiary/internal/testdb"
	"driverdiary/internal/trip"
)

const (
	t1 = `{"id":"t1","start":"2026-10-01T08:10:00+05:00","end":"2026-10-01T08:32:00+05:00","amount":2400,"payment":"card","commission":360}`
	t2 = `{"id":"t2","start":"2026-10-01T09:05:00+05:00","end":"2026-10-01T09:20:00+05:00","amount":1500,"payment":"cash","commission":225}`
)

func newServer(t *testing.T) *httptest.Server {
	loc, _ := trip.ParseOffset("+05:00")
	srv := httptest.NewServer(api.New(api.Options{Store: store.New(testdb.New(t), loc), Loc: loc}))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, srv *httptest.Server, method, path, body string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return res.StatusCode, out
}

func TestPostTwiceThenDay(t *testing.T) {
	srv := newServer(t)
	for _, c := range []struct {
		body string
		want int
	}{{t1, 201}, {t1, 200}, {t2, 201}, {t2, 200}} {
		if code, _ := do(t, srv, "POST", "/api/trips", c.body); code != c.want {
			t.Fatalf("POST want %d, got %d", c.want, code)
		}
	}

	code, body := do(t, srv, "GET", "/api/day?date=2026-10-01", "")
	if code != 200 {
		t.Fatalf("GET day: %d", code)
	}
	trips := body["trips"].([]any)
	if len(trips) != 2 {
		t.Fatalf("want 2 trips, got %d", len(trips))
	}
	s := body["summary"].(map[string]any)
	if s["trips"] != 2.0 || s["revenue"] != 3900.0 || s["commission"] != 585.0 || s["net"] != 3315.0 {
		t.Fatalf("summary: %v", s)
	}
	if s["cash"].(map[string]any)["amount"] != 1500.0 || s["card"].(map[string]any)["amount"] != 2400.0 {
		t.Fatalf("split: %v", s)
	}
}

func TestPostWithoutIDTwice(t *testing.T) {
	srv := newServer(t)
	body := `{"start":"2026-10-03T10:00:00+05:00","end":"2026-10-03T10:20:00+05:00","amount":1200,"payment":"cash","commission":180}`
	c1, r1 := do(t, srv, "POST", "/api/trips", body)
	c2, r2 := do(t, srv, "POST", "/api/trips", body)
	if c1 != 201 || c2 != 200 {
		t.Fatalf("codes %d %d", c1, c2)
	}
	if r1["trip"].(map[string]any)["id"] != r2["trip"].(map[string]any)["id"] {
		t.Fatal("ids must match")
	}
}

func TestErrors(t *testing.T) {
	srv := newServer(t)
	code, body := do(t, srv, "POST", "/api/trips",
		`{"start":"2026-10-01T08:10:00+05:00","end":"2026-10-01T08:10:00+05:00","amount":0,"payment":"card","commission":0}`)
	if code != 422 {
		t.Fatalf("want 422, got %d", code)
	}
	errs := body["errors"].(map[string]any)
	if _, ok := errs["amount"]; !ok {
		t.Fatalf("errors: %v", errs)
	}
	if _, ok := errs["end"]; !ok {
		t.Fatalf("errors: %v", errs)
	}

	do(t, srv, "POST", "/api/trips", t1)
	if code, _ := do(t, srv, "POST", "/api/trips", strings.Replace(t1, `"amount":2400`, `"amount":9999`, 1)); code != 409 {
		t.Fatalf("want 409, got %d", code)
	}
	if code, _ := do(t, srv, "POST", "/api/trips", "not json"); code != 422 {
		t.Fatalf("want 422, got %d", code)
	}
	if code, _ := do(t, srv, "GET", "/api/day?date=01.10.2026", ""); code != 400 {
		t.Fatalf("want 400, got %d", code)
	}
	if code, _ := do(t, srv, "GET", "/api/day", ""); code != 400 {
		t.Fatalf("want 400, got %d", code)
	}
}

func TestDaysAndSplitEndpoints(t *testing.T) {
	srv := newServer(t)
	do(t, srv, "POST", "/api/trips", t1)

	_, days := do(t, srv, "GET", "/api/days", "")
	if d := days["days"].([]any); len(d) != 1 || d[0] != "2026-10-01" || days["tz"] != "+05:00" {
		t.Fatalf("days: %v", days)
	}
	_, s := do(t, srv, "GET", "/api/summary?date=2026-10-01", "")
	if _, ok := s["trips"]; ok {
		t.Fatal("summary endpoint must not include trips")
	}
	_, tr := do(t, srv, "GET", "/api/trips?date=2026-10-02", "")
	if len(tr["trips"].([]any)) != 0 {
		t.Fatalf("want empty list, got %v", tr["trips"])
	}
}
