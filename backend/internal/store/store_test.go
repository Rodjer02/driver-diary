package store_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"driverdiary/internal/store"
	"driverdiary/internal/testdb"
	"driverdiary/internal/trip"
)

var loc, _ = trip.ParseOffset("+05:00")

func ts(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func t1() trip.Trip {
	return trip.Trip{ID: "t1", Start: ts("2026-10-01T08:10:00+05:00"), End: ts("2026-10-01T08:32:00+05:00"),
		Amount: 2400, Payment: trip.Card, Commission: 360}
}

func newStore(t *testing.T) *store.Store {
	return store.New(testdb.New(t), loc)
}

func dayCount(t *testing.T, s *store.Store, day string) []trip.Trip {
	t.Helper()
	from, to, _ := trip.DayRange(day, loc)
	trips, err := s.Between(context.Background(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	return trips
}

func mustAdd(t *testing.T, s *store.Store, tr trip.Trip) (trip.Trip, bool) {
	t.Helper()
	got, created, err := s.Add(context.Background(), tr)
	if err != nil {
		t.Fatal(err)
	}
	return got, created
}

func TestResendSameTripNoDuplicate(t *testing.T) {
	s := newStore(t)
	if _, created := mustAdd(t, s, t1()); !created {
		t.Fatal("first add must create")
	}
	if _, created := mustAdd(t, s, t1()); created {
		t.Fatal("resend must not create")
	}
	if n := len(dayCount(t, s, "2026-10-01")); n != 1 {
		t.Fatalf("want 1 trip, got %d", n)
	}
}

func TestSameContentDifferentIDNoDuplicate(t *testing.T) {
	s := newStore(t)
	mustAdd(t, s, t1())
	other := t1()
	other.ID = "other-id"
	// то же время, записанное в UTC
	other.Start, other.End = other.Start.UTC(), other.End.UTC()
	got, created := mustAdd(t, s, other)
	if created || got.ID != "t1" {
		t.Fatalf("want existing t1, got %s created=%v", got.ID, created)
	}
}

func TestSameIDDifferentDataConflict(t *testing.T) {
	s := newStore(t)
	mustAdd(t, s, t1())
	changed := t1()
	changed.Amount = 9999
	if _, _, err := s.Add(context.Background(), changed); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
}

func TestConcurrentResendsCreateOne(t *testing.T) {
	s := newStore(t)
	var wg sync.WaitGroup
	var mu sync.Mutex
	created := 0
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, c, err := s.Add(context.Background(), t1())
			if err != nil {
				t.Error(err)
				return
			}
			if c {
				mu.Lock()
				created++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if created != 1 {
		t.Fatalf("want exactly 1 created, got %d", created)
	}
	if n := len(dayCount(t, s, "2026-10-01")); n != 1 {
		t.Fatalf("want 1 trip, got %d", n)
	}
}

func TestDayBoundariesInDriverTZ(t *testing.T) {
	s := newStore(t)
	night := t1()
	night.ID, night.Start, night.End = "night", ts("2026-10-01T23:50:00+05:00"), ts("2026-10-02T00:20:00+05:00")
	utc := t1()
	// 20:00Z 1 октября == 01:00 2 октября по +05:00
	utc.ID, utc.Start, utc.End = "utc", ts("2026-10-01T20:00:00Z"), ts("2026-10-01T20:30:00Z")
	mustAdd(t, s, night)
	mustAdd(t, s, utc)

	if got := dayCount(t, s, "2026-10-01"); len(got) != 1 || got[0].ID != "night" {
		t.Fatalf("2026-10-01: %+v", got)
	}
	got := dayCount(t, s, "2026-10-02")
	if len(got) != 1 || got[0].ID != "utc" {
		t.Fatalf("2026-10-02: %+v", got)
	}
	if got[0].Start.Format(time.RFC3339) != "2026-10-02T01:00:00+05:00" {
		t.Fatalf("time must be returned in driver tz: %s", got[0].Start)
	}

	days, err := s.Days(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(days) != 2 || days[0] != "2026-10-01" || days[1] != "2026-10-02" {
		t.Fatalf("days: %v", days)
	}
}
