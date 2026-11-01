// Package ratelimit limita tentativas por chave (ex.: IP) em janela fixa, na memória.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter permite no máximo max eventos por chave a cada window.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	now    func() time.Time
	hits   map[string]*bucket
}

type bucket struct {
	start time.Time
	n     int
}

// New cria o limitador.
func New(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, now: time.Now, hits: map[string]*bucket{}}
}

// Allow registra uma tentativa e informa se ainda está dentro do limite.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	if len(l.hits) > 10000 { // evita crescimento sem fim
		for k, b := range l.hits {
			if now.Sub(b.start) > l.window {
				delete(l.hits, k)
			}
		}
	}
	b := l.hits[key]
	if b == nil || now.Sub(b.start) > l.window {
		b = &bucket{start: now}
		l.hits[key] = b
	}
	b.n++
	return b.n <= l.max
}
