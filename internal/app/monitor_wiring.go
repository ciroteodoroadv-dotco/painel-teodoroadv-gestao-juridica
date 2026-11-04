package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/monitor"
	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/processos"
)

// DataJud é a configuração do monitor do DataJud. O zero-valor é "desligado": sem rede, sem agendador.
type DataJud struct {
	Ativo        bool
	BaseURL      string // vazio = endereço oficial
	PermitirHTTP bool   // só para teste
	Chave        func() string
	Intervalo    time.Duration
	Pausa        time.Duration // entre consultas
	AlertaHoras  int
	Transport    http.RoundTripper // só para teste
}

// WithDataJud define a configuração do monitor do DataJud.
func WithDataJud(d DataJud) Option { return func(o *options) { o.datajud = d } }

// WithColetorSigiloso troca o cliente do DJEN por processo sigiloso (teste).
func WithColetorSigiloso(c intimacoes.ColetorPorNumero) Option {
	return func(o *options) { o.sigiloso = c }
}

type monProcessos struct{ p *processos.Service }

func (a monProcessos) Monitoraveis(ctx context.Context) ([]monitor.ProcessoRef, error) {
	ls, err := a.p.ListarParaMonitor(ctx)
	out := make([]monitor.ProcessoRef, len(ls))
	for i, m := range ls {
		out[i] = monitor.ProcessoRef{ID: m.ID, Numero: m.Numero, CriadoPor: m.CriadoPor}
	}
	return out, err
}

func (a monProcessos) Ref(ctx context.Context, id string) (monitor.ProcessoRef, bool, error) {
	m, ok, err := a.p.RefParaMonitor(ctx, id)
	return monitor.ProcessoRef{ID: m.ID, Numero: m.Numero, CriadoPor: m.CriadoPor}, ok, err
}

type monOABs struct{ s *intimacoes.Service }

func (a monOABs) Obter(ctx context.Context, id string) (monitor.OABRef, error) {
	o, err := a.s.ObterOAB(ctx, id)
	return monitor.OABRef{ID: o.ID, Numero: o.Numero, UF: o.UF, AutorizacaoBase: o.AutorizacaoBase, AutorizacaoEm: o.AutorizacaoEm, Ativa: o.Ativa}, err
}

type monRegistro struct{ s *intimacoes.Service }

func (a monRegistro) RegistrarNaTx(ctx context.Context, tx pgx.Tx, n monitor.NovaIntimacao) (string, bool, error) {
	return a.s.RegistrarNaTx(ctx, tx, intimacoes.Nova(n))
}

type monSigilosos struct{ c intimacoes.ColetorPorNumero }

func (a monSigilosos) ColetarPorNumero(ctx context.Context, oab, uf, numero string, desde, ate time.Time) ([]monitor.ComunicacaoSigilosa, error) {
	ls, err := a.c.ColetarPorNumero(ctx, oab, uf, numero, desde, ate)
	out := make([]monitor.ComunicacaoSigilosa, len(ls))
	for i, c := range ls {
		out[i] = monitor.ComunicacaoSigilosa{Hash: c.Hash(), Tribunal: c.Tribunal, Orgao: c.Orgao, Processo: c.Processo, Tipo: c.Tipo,
			Texto: c.Texto, Resumo: c.Resumo(), Link: c.Link, Data: c.Data}
	}
	return out, err
}

// montarMonitor cria o serviço do monitor. O cliente do DataJud é construído sempre (valida o https), mas só é chamado com a
// coleta ATIVA e com chave: desligada, o serviço registra "bloqueado" e não faz nenhuma requisição.
func montarMonitor(a *auth.Auth, pool *pgxpool.Pool, proc *processos.Service, intim *intimacoes.Service, o *options) (*monitor.Service, error) {
	cli, err := datajud.Novo(datajud.Config{BaseURL: o.datajud.BaseURL, PermitirHTTP: o.datajud.PermitirHTTP, Chave: o.datajud.Chave,
		Intervalo: o.datajud.Pausa, Transport: o.datajud.Transport})
	if err != nil {
		return nil, err
	}
	var sig monitor.Sigilosos
	switch {
	case o.sigiloso != nil:
		sig = monSigilosos{c: o.sigiloso}
	case o.djen.Ativo:
		if dc, ok := intim.Coletor.(*djen.Cliente); ok {
			sig = monSigilosos{c: intimacoes.ClienteSigiloso{C: dc}}
		}
	}
	svc := &monitor.Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Cliente: cli, Processos: monProcessos{p: proc}, OABs: monOABs{s: intim},
		Registro: monRegistro{s: intim}, Sigilosos: sig, Now: o.agora,
		Cfg: monitor.Config{Ativo: o.datajud.Ativo, Intervalo: o.datajud.Intervalo, AlertaHoras: o.datajud.AlertaHoras}}
	a.AddFaixa(func(ctx context.Context, _ *auth.Session) (auth.Faixa, error) {
		forte, texto, ativo, err := svc.Alerta(ctx)
		if err != nil || !ativo {
			return auth.Faixa{}, err
		}
		return auth.Faixa{Strong: forte, Text: texto}, nil
	})
	return svc, nil
}

func (a monOABs) Listar(ctx context.Context) ([]monitor.OABRef, error) {
	ls, err := a.s.ListarOABs(ctx)
	out := make([]monitor.OABRef, len(ls))
	for i, o := range ls {
		out[i] = monitor.OABRef{ID: o.ID, Numero: o.Numero, UF: o.UF, AutorizacaoBase: o.AutorizacaoBase, AutorizacaoEm: o.AutorizacaoEm, Ativa: o.Ativa}
	}
	return out, err
}
