// seed загружает поездки из JSON-файла в БД. Повторный запуск безопасен: дубли не создаются.
//
//	DATABASE_URL=... go run ./cmd/seed ../data/trips.json
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"driverdiary/internal/app"
	"driverdiary/internal/trip"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: seed <trips.json>")
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		log.Fatalf("%s: ожидается JSON-массив поездок: %v", os.Args[1], err)
	}

	ctx := context.Background()
	a, err := app.FromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer a.Pool.Close()

	var created, skipped, failed int
	for i, item := range items {
		t, err := trip.Parse(item)
		if err == nil {
			var isNew bool
			_, isNew, err = a.Store.Add(ctx, t)
			if isNew {
				created++
			} else if err == nil {
				skipped++
			}
		}
		if err != nil {
			failed++
			log.Printf("запись %d: %v", i, err)
		}
	}
	fmt.Printf("добавлено: %d, уже были: %d, ошибок: %d\n", created, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
}
