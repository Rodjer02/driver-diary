// Точка входа для Vercel (Go Runtime). Все пути переписываются сюда через vercel.json.
// Пул соединений создаётся один раз на инстанс функции.
package handler

import (
	"context"
	"log"
	"net/http"
	"sync"

	"driverdiary/internal/app"
)

var (
	once    sync.Once
	handler http.Handler
	initErr error
)

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(func() {
		a, err := app.FromEnv(context.Background())
		if err != nil {
			initErr = err
			return
		}
		handler = a.Handler
	})
	if initErr != nil {
		log.Printf("init: %v", initErr)
		http.Error(w, `{"error":"сервер не настроен"}`, http.StatusInternalServerError)
		return
	}
	handler.ServeHTTP(w, r)
}
