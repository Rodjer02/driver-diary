package trip

import (
	"encoding/json"
	"fmt"
	"testing"
)

var (
	t1 = map[string]any{"id": "t1", "start": "2026-10-01T08:10:00+05:00", "end": "2026-10-01T08:32:00+05:00",
		"amount": 2400, "payment": "card", "commission": 360}
	t2 = map[string]any{"id": "t2", "start": "2026-10-01T09:05:00+05:00", "end": "2026-10-01T09:20:00+05:00",
		"amount": 1500, "payment": "cash", "commission": 225}
)

func with(base map[string]any, kv ...any) map[string]any {
	m := map[string]any{}
	for k, v := range base {
		m[k] = v
	}
	for i := 0; i < len(kv); i += 2 {
		if kv[i+1] == nil {
			delete(m, kv[i].(string))
		} else {
			m[kv[i].(string)] = kv[i+1]
		}
	}
	return m
}

func mustParse(t *testing.T, m map[string]any) Trip {
	t.Helper()
	b, _ := json.Marshal(m)
	tr, err := Parse(b)
	if err != nil {
		t.Fatalf("Parse(%s): %v", b, err)
	}
	return tr
}

func TestSummarizeExampleDay(t *testing.T) {
	s := Summarize([]Trip{mustParse(t, t1), mustParse(t, t2)})
	want := Summary{Trips: 2, Revenue: 3900, Commission: 585, Net: 3315,
		Cash: PaymentTotals{1, 1500}, Card: PaymentTotals{1, 2400}}
	if s != want {
		t.Fatalf("got %+v, want %+v", s, want)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	if s := Summarize(nil); s != (Summary{}) {
		t.Fatalf("got %+v", s)
	}
}

func TestSummarizeInvariants(t *testing.T) {
	var trips []Trip
	var net int64
	for i := range 7 {
		p := "card"
		if i%3 != 0 {
			p = "cash"
		}
		tr := mustParse(t, with(t1, "id", fmt.Sprintf("x%d", i), "amount", 100*(i+1), "commission", i, "payment", p,
			"start", fmt.Sprintf("2026-10-01T1%d:00:00+05:00", i), "end", fmt.Sprintf("2026-10-01T1%d:30:00+05:00", i)))
		trips = append(trips, tr)
		net += tr.Net()
	}
	s := Summarize(trips)
	if s.Cash.Amount+s.Card.Amount != s.Revenue || s.Cash.Count+s.Card.Count != s.Trips || s.Net != net {
		t.Fatalf("invariants broken: %+v", s)
	}
}

func TestValidation(t *testing.T) {
	cases := []struct {
		name  string
		data  map[string]any
		field string
	}{
		{"amount zero", with(t1, "amount", 0), "amount"},
		{"amount negative", with(t1, "amount", -5), "amount"},
		{"amount string", with(t1, "amount", "2400"), "amount"},
		{"amount bool", with(t1, "amount", true), "amount"},
		{"amount fraction", with(t1, "amount", 10.5), "amount"},
		{"end equals start", with(t1, "end", t1["start"]), "end"},
		{"end before start", with(t1, "end", "2026-10-01T08:00:00+05:00"), "end"},
		{"end before start other tz", with(t1, "end", "2026-10-01T03:05:00Z"), "end"},
		{"start without tz", with(t1, "start", "2026-10-01T08:10:00"), "start"},
		{"payment unknown", with(t1, "payment", "crypto"), "payment"},
		{"commission negative", with(t1, "commission", -1), "commission"},
		{"commission above amount", with(t1, "commission", 5000), "commission"},
		{"missing end", with(t1, "end", nil), "end"},
		{"empty id", with(t1, "id", " "), "id"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, _ := json.Marshal(c.data)
			_, err := Parse(b)
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("want ValidationError, got %v", err)
			}
			if _, ok := ve.Errors[c.field]; !ok {
				t.Fatalf("want error on %q, got %v", c.field, ve.Errors)
			}
		})
	}
}

func TestValidationCollectsAllErrors(t *testing.T) {
	b, _ := json.Marshal(with(t1, "amount", 0, "end", t1["start"]))
	_, err := Parse(b)
	ve := err.(*ValidationError)
	if len(ve.Errors) != 2 {
		t.Fatalf("want 2 errors, got %v", ve.Errors)
	}
}

func TestValidAcrossTimezones(t *testing.T) {
	// начало 08:10+05:00 == 03:10Z, окончание 03:20Z позже — валидно
	mustParse(t, with(t1, "end", "2026-10-01T03:20:00Z"))
}

func TestNotJSONObject(t *testing.T) {
	for _, body := range []string{"", "[]", "{", "42"} {
		if _, err := Parse([]byte(body)); err == nil {
			t.Fatalf("Parse(%q) want error", body)
		}
	}
}

func TestAutoIDIsDeterministic(t *testing.T) {
	a := mustParse(t, with(t2, "id", nil))
	b := mustParse(t, with(t2, "id", nil))
	if a.ID == "" || a.ID != b.ID {
		t.Fatalf("ids: %q %q", a.ID, b.ID)
	}
}

func TestFingerprintIgnoresIDAndOffset(t *testing.T) {
	a := mustParse(t, t1)
	b := mustParse(t, with(t1, "id", "other", "start", "2026-10-01T03:10:00Z", "end", "2026-10-01T03:32:00Z"))
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("same trip must have same fingerprint")
	}
	c := mustParse(t, with(t1, "amount", 2401))
	if a.Fingerprint() == c.Fingerprint() {
		t.Fatal("different trips must differ")
	}
}

func TestDayRange(t *testing.T) {
	loc, err := ParseOffset("+05:00")
	if err != nil {
		t.Fatal(err)
	}
	from, to, err := DayRange("2026-10-01", loc)
	if err != nil {
		t.Fatal(err)
	}
	if got := from.UTC().Format("2006-01-02T15:04Z"); got != "2026-09-30T19:00Z" {
		t.Fatalf("from = %s", got)
	}
	if to.Sub(from).Hours() != 24 {
		t.Fatalf("to = %s", to)
	}
	if _, _, err := DayRange("01.10.2026", loc); err == nil {
		t.Fatal("want error")
	}
	if _, err := ParseOffset("Asia/Almaty"); err == nil {
		t.Fatal("want error")
	}
}
