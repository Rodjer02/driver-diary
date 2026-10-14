// Package api — HTTP API дневника смен.
//
//	GET  /api/days                  -> {"days": [...], "tz": "+05:00"}
//	GET  /api/day?date=YYYY-MM-DD   -> {"date", "summary", "trips"}
//	GET  /api/trips?date=YYYY-MM-DD -> {"date", "trips"}
//	GET  /api/summary?date=...      -> {"date", "summary"}
//	POST /api/trips                 -> 201 создана | 200 уже была | 409 конфликт id | 422 ошибки валидации
//	GET  /healthz                   -> 200
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"driverdiary/internal/store"
	"driverdiary/internal/trip"
)

const maxBody = 64 << 10

type Store interface {
	Add(ctx context.Context, t trip.Trip) (trip.Trip, bool, error)
	Between(ctx context.Context, from, to time.Time) ([]trip.Trip, error)
	Days(ctx context.Context) ([]string, error)
}

type Options struct {
	Store Store
	Loc   *time.Location
	// CORSOrigin — разрешённый Origin для прямых запросов из браузера. Пусто — CORS выключен.
	CORSOrigin string
}

func New(o Options) http.Handler {
	h := &handler{store: o.Store, loc: o.Loc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("GET /api/days", h.days)
	mux.HandleFunc("GET /api/day", h.day(true, true))
	mux.HandleFunc("GET /api/trips", h.day(true, false))
	mux.HandleFunc("GET /api/summary", h.day(false, true))
	mux.HandleFunc("POST /api/trips", h.addTrip)
	return withCORS(o.CORSOrigin, mux)
}

type handler struct {
	store Store
	loc   *time.Location
}

func (h *handler) days(w http.ResponseWriter, r *http.Request) {
	days, err := h.store.Days(r.Context())
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": days, "tz": h.loc.String()})
}

func (h *handler) day(withTrips, withSummary bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		day := r.URL.Query().Get("date")
		if day == "" {
			writeError(w, http.StatusBadRequest, "параметр date обязателен (YYYY-MM-DD)", nil)
			return
		}
		from, to, err := trip.DayRange(day, h.loc)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error(), nil)
			return
		}
		trips, err := h.store.Between(r.Context(), from, to)
		if err != nil {
			internalError(w, err)
			return
		}
		resp := map[string]any{"date": day}
		if withTrips {
			resp["trips"] = trips
		}
		if withSummary {
			resp["summary"] = trip.Summarize(trips)
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func (h *handler) addTrip(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "слишком большое тело запроса", nil)
		return
	}
	t, err := trip.Parse(body)
	var ve *trip.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusUnprocessableEntity, "ошибка валидации", ve.Errors)
		return
	}
	saved, created, err := h.store.Add(r.Context(), t)
	switch {
	case errors.Is(err, store.ErrConflict):
		writeError(w, http.StatusConflict, err.Error(), nil)
		return
	case err != nil:
		internalError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, map[string]any{"trip": saved, "created": created})
}

func withCORS(origin string, next http.Handler) http.Handler {
	if origin == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Add("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string, fields map[string]string) {
	resp := map[string]any{"error": msg}
	if fields != nil {
		resp["errors"] = fields
	}
	writeJSON(w, status, resp)
}

func internalError(w http.ResponseWriter, err error) {
	log.Printf("internal error: %v", err)
	writeError(w, http.StatusInternalServerError, "внутренняя ошибка сервера", nil)
}
