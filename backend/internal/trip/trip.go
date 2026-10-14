// Package trip — доменная логика: поездка, её валидация, сводка за день и границы дня.
package trip

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

type Payment string

const (
	Cash Payment = "cash"
	Card Payment = "card"
)

type Trip struct {
	ID         string    `json:"id"`
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Amount     int64     `json:"amount"`
	Payment    Payment   `json:"payment"`
	Commission int64     `json:"commission"`
}

func (t Trip) Net() int64 { return t.Amount - t.Commission }

// Fingerprint — хэш содержимого поездки. Одинаковые поездки дают одинаковый отпечаток
// независимо от id и от того, в каком часовом поясе записано время.
func (t Trip) Fingerprint() string {
	raw := strings.Join([]string{
		t.Start.UTC().Format(time.RFC3339Nano),
		t.End.UTC().Format(time.RFC3339Nano),
		strconv.FormatInt(t.Amount, 10),
		string(t.Payment),
		strconv.FormatInt(t.Commission, 10),
	}, "|")
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// In возвращает копию поездки со временем в указанном поясе.
func (t Trip) In(loc *time.Location) Trip {
	t.Start = t.Start.In(loc)
	t.End = t.End.In(loc)
	return t
}

// ValidationError содержит все ошибки сразу: поле -> сообщение.
type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Errors))
	for k, v := range e.Errors {
		parts = append(parts, k+": "+v)
	}
	return strings.Join(parts, "; ")
}

var requiredFields = []string{"start", "end", "amount", "payment", "commission"}

// Parse разбирает и проверяет JSON поездки. Если id не прислан, он выводится из содержимого,
// чтобы повторная отправка той же поездки не создавала дубль.
func Parse(body []byte) (Trip, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return Trip{}, &ValidationError{Errors: map[string]string{"_": "тело запроса — не JSON"}}
	}
	data, ok := raw.(map[string]any)
	if !ok {
		return Trip{}, &ValidationError{Errors: map[string]string{"_": "ожидается JSON-объект"}}
	}

	errs := map[string]string{}
	for _, f := range requiredFields {
		if _, ok := data[f]; !ok {
			errs[f] = "обязательное поле"
		}
	}

	var t Trip
	var okStart, okEnd, okAmount, okCommission bool
	if v, ok := data["start"]; ok {
		t.Start, okStart = parseTime(v, "start", errs)
	}
	if v, ok := data["end"]; ok {
		t.End, okEnd = parseTime(v, "end", errs)
	}
	if v, ok := data["amount"]; ok {
		t.Amount, okAmount = parseMoney(v, "amount", errs)
	}
	if v, ok := data["commission"]; ok {
		t.Commission, okCommission = parseMoney(v, "commission", errs)
	}
	if v, ok := data["payment"]; ok {
		p, _ := v.(string)
		t.Payment = Payment(p)
		if t.Payment != Cash && t.Payment != Card {
			errs["payment"] = "допустимо: cash, card"
		}
	}
	if v, ok := data["id"]; ok && v != nil {
		id, isStr := v.(string)
		if !isStr || strings.TrimSpace(id) == "" {
			errs["id"] = "id должен быть непустой строкой"
		}
		t.ID = id
	}

	if okStart && okEnd && !t.End.After(t.Start) {
		errs["end"] = "окончание должно быть позже начала"
	}
	if okAmount && t.Amount <= 0 {
		errs["amount"] = "сумма должна быть больше 0"
	}
	if okCommission {
		switch {
		case t.Commission < 0:
			errs["commission"] = "комиссия не может быть отрицательной"
		case okAmount && t.Amount > 0 && t.Commission > t.Amount:
			errs["commission"] = "комиссия не может превышать сумму"
		}
	}

	if len(errs) > 0 {
		return Trip{}, &ValidationError{Errors: errs}
	}
	if t.ID == "" {
		t.ID = "auto-" + t.Fingerprint()[:12]
	}
	return t, nil
}

func parseTime(v any, field string, errs map[string]string) (time.Time, bool) {
	s, ok := v.(string)
	if !ok {
		errs[field] = "ожидается строка ISO 8601"
		return time.Time{}, false
	}
	// RFC3339 требует пояс (Z или ±hh:mm) — время без пояса отклоняется
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		errs[field] = "нужно время ISO 8601 с поясом, например 2026-10-01T08:10:00+05:00"
		return time.Time{}, false
	}
	return t, true
}

func parseMoney(v any, field string, errs map[string]string) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		errs[field] = "ожидается число"
		return 0, false
	}
	if i, err := n.Int64(); err == nil {
		return i, true
	}
	f, err := n.Float64()
	if err != nil || f != math.Trunc(f) || math.Abs(f) > 1e15 {
		errs[field] = "ожидается целое число"
		return 0, false
	}
	return int64(f), true
}

type PaymentTotals struct {
	Count  int   `json:"count"`
	Amount int64 `json:"amount"`
}

type Summary struct {
	Trips      int           `json:"trips"`
	Revenue    int64         `json:"revenue"`
	Commission int64         `json:"commission"`
	Net        int64         `json:"net"` // «на руки» = выручка − комиссия
	Cash       PaymentTotals `json:"cash"`
	Card       PaymentTotals `json:"card"`
}

func Summarize(trips []Trip) Summary {
	var s Summary
	for _, t := range trips {
		s.Trips++
		s.Revenue += t.Amount
		s.Commission += t.Commission
		bucket := &s.Card
		if t.Payment == Cash {
			bucket = &s.Cash
		}
		bucket.Count++
		bucket.Amount += t.Amount
	}
	s.Net = s.Revenue - s.Commission
	return s
}

// DayRange — полуинтервал [from, to) дня в поясе водителя.
// Поездка относится к дню своего начала.
func DayRange(day string, loc *time.Location) (from, to time.Time, err error) {
	from, err = time.ParseInLocation(time.DateOnly, day, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверный формат даты, нужен YYYY-MM-DD")
	}
	return from, from.AddDate(0, 0, 1), nil
}

// ParseOffset разбирает фиксированное смещение вида "+05:00".
func ParseOffset(s string) (*time.Location, error) {
	t, err := time.Parse("-07:00", s)
	if err != nil {
		return nil, fmt.Errorf("неверный часовой пояс %q, нужен формат +05:00", s)
	}
	_, offset := t.Zone()
	return time.FixedZone(s, offset), nil
}
