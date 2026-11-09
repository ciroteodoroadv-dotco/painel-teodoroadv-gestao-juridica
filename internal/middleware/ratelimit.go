package middleware

import (
"net"
"net/http"
"sync"
"time"
)

type visitor struct {
lastSeen time.Time
tokens   int
}

type RateLimiter struct {
mu       sync.Mutex
visitors map[string]*visitor
rate     int           // Requisições por janela
window   time.Duration // Janela de tempo
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
rl := &RateLimiter{
visitors: make(map[string]*visitor),
rate:     rate,
window:   window,
}
go rl.cleanup()
return rl
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
ip, _, err := net.SplitHostPort(r.RemoteAddr)
if err != nil {
ip = r.RemoteAddr
}

rl.mu.Lock()
v, exists := rl.visitors[ip]
now := time.Now()

if !exists {
rl.visitors[ip] = &visitor{lastSeen: now, tokens: rl.rate - 1}
rl.mu.Unlock()
next.ServeHTTP(w, r)
return
}

if now.Sub(v.lastSeen) > rl.window {
v.tokens = rl.rate - 1
v.lastSeen = now
rl.mu.Unlock()
next.ServeHTTP(w, r)
return
}

if v.tokens <= 0 {
rl.mu.Unlock()
http.Error(w, "Limite de requisições excedido. Tente novamente em instantes.", http.StatusTooManyRequests)
return
}

v.tokens--
v.lastSeen = now
rl.mu.Unlock()

next.ServeHTTP(w, r)
})
}

func (rl *RateLimiter) cleanup() {
for {
time.Sleep(5 * time.Minute)
rl.mu.Lock()
for ip, v := range rl.visitors {
if time.Since(v.lastSeen) > 10*time.Minute {
delete(rl.visitors, ip)
}
}
rl.mu.Unlock()
}
}
