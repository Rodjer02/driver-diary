// Package store хранит поездки в PostgreSQL. Защита от дублей — на уровне БД:
// PRIMARY KEY (id) + UNIQUE (fingerprint) + INSERT ... ON CONFLICT DO NOTHING.
package store

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"driverdiary/internal/trip"
)

//go:embed schema.sql
var schema string

// ErrConflict — поездка с таким id уже есть, но с другими данными.
var ErrConflict = errors.New("поездка с таким id уже существует с другими данными")

type Store struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

// New создаёт хранилище. loc — пояс водителя: в нём отдаётся время и считаются границы дней.
func New(pool *pgxpool.Pool, loc *time.Location) *Store {
	return &Store{pool: pool, loc: loc}
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	return err
}

const columns = "id, start_at, end_at, amount, payment, commission, fingerprint"

// Add добавляет поездку. Возвращает (поездка, создана_ли).
// Повтор той же поездки (тот же id или то же содержимое) возвращает существующую запись.
func (s *Store) Add(ctx context.Context, t trip.Trip) (trip.Trip, bool, error) {
	fp := t.Fingerprint()
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO trips (`+columns+`) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING`,
		t.ID, t.Start, t.End, t.Amount, string(t.Payment), t.Commission, fp)
	if err != nil {
		return trip.Trip{}, false, fmt.Errorf("insert trip: %w", err)
	}
	if tag.RowsAffected() == 1 {
		return t.In(s.loc), true, nil
	}

	// Конфликт по id или по отпечатку. Строки не удаляются, поэтому одна из выборок найдёт запись.
	existing, existingFP, err := s.one(ctx, "id = $1", t.ID)
	if err == nil {
		if existingFP != fp {
			return trip.Trip{}, false, ErrConflict
		}
		return existing, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, false, err
	}
	existing, _, err = s.one(ctx, "fingerprint = $1", fp)
	if err != nil {
		return trip.Trip{}, false, fmt.Errorf("find duplicate: %w", err)
	}
	return existing, false, nil
}

func (s *Store) one(ctx context.Context, where string, arg any) (trip.Trip, string, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+columns+` FROM trips WHERE `+where, arg)
	var t trip.Trip
	var fp string
	if err := row.Scan(&t.ID, &t.Start, &t.End, &t.Amount, &t.Payment, &t.Commission, &fp); err != nil {
		return trip.Trip{}, "", err
	}
	return t.In(s.loc), fp, nil
}

// Between — поездки, начавшиеся в [from, to), по времени начала.
func (s *Store) Between(ctx context.Context, from, to time.Time) ([]trip.Trip, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, start_at, end_at, amount, payment, commission FROM trips
		 WHERE start_at >= $1 AND start_at < $2 ORDER BY start_at, id`, from, to)
	if err != nil {
		return nil, err
	}
	trips, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (trip.Trip, error) {
		var t trip.Trip
		err := r.Scan(&t.ID, &t.Start, &t.End, &t.Amount, &t.Payment, &t.Commission)
		return t.In(s.loc), err
	})
	if err != nil {
		return nil, err
	}
	if trips == nil {
		trips = []trip.Trip{}
	}
	return trips, nil
}

// Days — даты (YYYY-MM-DD в поясе водителя), в которые есть поездки.
func (s *Store) Days(ctx context.Context) ([]string, error) {
	_, offset := time.Now().In(s.loc).Zone()
	rows, err := s.pool.Query(ctx,
		`SELECT DISTINCT to_char((start_at AT TIME ZONE 'UTC') + make_interval(secs => $1), 'YYYY-MM-DD') AS d
		 FROM trips ORDER BY d`, offset)
	if err != nil {
		return nil, err
	}
	days, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	if days == nil {
		days = []string{}
	}
	return days, nil
}
