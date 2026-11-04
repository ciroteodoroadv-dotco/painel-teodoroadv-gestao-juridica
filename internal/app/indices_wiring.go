package app

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/indices"
	"painelteodoro/internal/indices/bcb"
	"painelteodoro/internal/platform/coletas"
)

// Indices é a configuração dos índices do BCB no rodapé. O zero-valor é "desligado": sem rede, sem agendador.
type Indices struct {
	Ativo        bool
	BaseURL      string // vazio = endereço oficial
	PermitirHTTP bool   // só para teste
	Intervalo    time.Duration
	ValidadeDias int
	Pausa        time.Duration // entre séries
	Transport    http.RoundTripper
}

// WithIndices define a configuração dos índices do BCB.
func WithIndices(i Indices) Option { return func(o *options) { o.indices = i } }

// montarIndices cria o serviço e liga o rodapé. O cliente do BCB é construído sempre (valida o https), mas só é chamado
// com a coleta ATIVA: desligada, o serviço registra "bloqueado" e não faz nenhuma requisição.
func montarIndices(a *auth.Auth, rt *auth.Router, pool *pgxpool.Pool, o *options) (*indices.Service, error) {
	pausa := o.indices.Pausa
	if pausa <= 0 {
		pausa = time.Second
	}
	cli, err := bcb.Novo(bcb.Config{BaseURL: o.indices.BaseURL, PermitirHTTP: o.indices.PermitirHTTP, Pausa: pausa, Transport: o.indices.Transport})
	if err != nil {
		return nil, err
	}
	svc := &indices.Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Fonte: cli, Ativo: o.indices.Ativo,
		Intervalo: o.indices.Intervalo, ValidadeDias: o.indices.ValidadeDias, Now: o.agora}
	(&indices.Handler{A: a, S: svc}).Register(rt)
	return svc, nil
}
