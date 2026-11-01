// Package health expõe /healthz (vivo) e /readyz (pronto: banco responde).
package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Pinger é satisfeito pelo pool do pgx.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Register adiciona as duas rotas ao mux.
func Register(mux *http.ServeMux, db Pinger, log *slog.Logger) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if err := db.Ping(ctx); err != nil {
			log.Error("banco indisponível", "rota", "/readyz")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("indisponível"))
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
}
