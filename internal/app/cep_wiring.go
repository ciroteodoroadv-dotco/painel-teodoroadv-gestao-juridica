package app

import (
	"net/http"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/cep"
	"painelteodoro/internal/platform/ratelimit"
)

// CEP é a configuração da consulta de CEP (ViaCEP). O zero-valor é "desligado": a rota responde que está desligada, sem rede.
type CEP struct {
	Ativo           bool
	BaseURL         string // vazio = endereço oficial
	PermitirHTTP    bool   // só para teste
	LimitePorMinuto int    // por usuário; 0 = 20
	Transport       http.RoundTripper
}

// WithCEP define a configuração da consulta de CEP.
func WithCEP(c CEP) Option { return func(o *options) { o.cep = c } }

// montarCEP cria o cliente (valida o https; só é chamado com a consulta ATIVA) e registra a rota.
func montarCEP(a *auth.Auth, rt *auth.Router, o *options) error {
	cli, err := cep.NovoCliente(cep.Config{BaseURL: o.cep.BaseURL, PermitirHTTP: o.cep.PermitirHTTP, Transport: o.cep.Transport})
	if err != nil {
		return err
	}
	lim := o.cep.LimitePorMinuto
	if lim <= 0 {
		lim = 20
	}
	(&cep.Handler{A: a, Ativo: o.cep.Ativo, B: cli, Limite: ratelimit.New(lim, time.Minute), Agora: o.agora}).Register(rt)
	return nil
}
