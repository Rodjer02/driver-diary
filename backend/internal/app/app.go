// Package app собирает приложение из переменных окружения:
//
//	DATABASE_URL — строка подключения PostgreSQL (обязательно)
//	DRIVER_TZ    — пояс водителя для границ дня, по умолчанию +05:00
//	CORS_ORIGIN  — Origin фронтенда при прямых запросах из браузера (необязательно)
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"driverdiary/internal/api"
	"driverdiary/internal/store"
	"driverdiary/internal/trip"
)

type App struct {
	Pool    *pgxpool.Pool
	Store   *store.Store
	Handler http.Handler
}

func FromEnv(ctx context.Context) (*App, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL не задан")
	}
	tz := os.Getenv("DRIVER_TZ")
	if tz == "" {
		tz = "+05:00"
	}
	loc, err := trip.ParseOffset(tz)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := store.Migrate(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	st := store.New(pool, loc)
	return &App{
		Pool:    pool,
		Store:   st,
		Handler: api.New(api.Options{Store: st, Loc: loc, CORSOrigin: os.Getenv("CORS_ORIGIN")}),
	}, nil
}
