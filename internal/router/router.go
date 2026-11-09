package router

import (
"database/sql"
"net/http"

"painelteodoro/internal/handler/live"
"painelteodoro/internal/middleware"
"painelteodoro/internal/sse"
"time"
)

type Config struct {
DB  *sql.DB
Hub *sse.Hub
}

func NewRouter(cfg Config) http.Handler {
mux := http.NewServeMux()

searchHandler := live.NewSearchHandler(cfg.DB)

mux.Handle("/events", cfg.Hub)
mux.HandleFunc("/api/processos/busca", searchHandler.BuscarProcessosHTMX)

fileServer := http.FileServer(http.Dir("./static"))
mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

// Aplica Rate Limiter (60 requisições por minuto) e Secure Headers
limiter := middleware.NewRateLimiter(60, 1*time.Minute)

var handler http.Handler = mux
handler = limiter.Limit(handler)
handler = middleware.SecureHeaders(handler)

return handler
}
