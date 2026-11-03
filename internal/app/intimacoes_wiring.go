package app

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/agenda"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/intimacoes"
	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/platform/coletas"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// DJEN é a configuração da coleta de intimações. O zero-valor é "desligada": sem rede, sem agendador.
type DJEN struct {
	Ativo       bool
	BaseURL     string // vazio = endereço oficial
	Intervalo   time.Duration
	AlertaHoras int
	Transport   http.RoundTripper // só para teste
}

// Servicos devolve ao chamador os serviços montados (o main liga o agendador; os testes chamam direto).
type Servicos struct {
	Intimacoes *intimacoes.Service
	Coletas    *coletas.Store
}

type options struct {
	djen           DJEN
	coletor        intimacoes.Coletor
	agora          func() time.Time
	envolveTarefas func(intimacoes.TarefaAutomatica) intimacoes.TarefaAutomatica
	out            *Servicos
}

// Option ajusta RegisterDomains.
type Option func(*options)

// WithDJEN define a configuração da coleta do DJEN.
func WithDJEN(d DJEN) Option { return func(o *options) { o.djen = d } }

// WithColetor troca o cliente do DJEN (teste).
func WithColetor(c intimacoes.Coletor) Option { return func(o *options) { o.coletor = c } }

// WithRelogio injeta o relógio do serviço de intimações (teste).
func WithRelogio(f func() time.Time) Option { return func(o *options) { o.agora = f } }

// WithTarefaAutomatica envolve a criação da tarefa de análise (teste: forçar falha).
func WithTarefaAutomatica(f func(intimacoes.TarefaAutomatica) intimacoes.TarefaAutomatica) Option {
	return func(o *options) { o.envolveTarefas = f }
}

// WithServicos pede de volta os serviços montados.
func WithServicos(s *Servicos) Option { return func(o *options) { o.out = s } }

// intimTarefas liga Intimações a Tarefas: cria a tarefa de análise dentro da transação e lê o estado dela.
type intimTarefas struct{ s *tarefas.Service }

func (a intimTarefas) CriarAnaliseDeIntimacao(ctx context.Context, tx pgx.Tx, in intimacoes.AnaliseIn) (string, error) {
	return a.s.CriarAnaliseDeIntimacao(ctx, tx, tarefas.AnaliseIn(in))
}

func (a intimTarefas) PorIntimacoes(ctx context.Context, ids []string) (map[string]intimacoes.TarefaResumo, error) {
	m, err := a.s.PorIntimacoes(ctx, ids)
	out := make(map[string]intimacoes.TarefaResumo, len(m))
	for id, r := range m {
		out[id] = intimacoes.TarefaResumo{Titulo: r.Titulo, Estado: r.Estado}
	}
	return out, err
}

// intimProcessos liga Intimações a Processos: o número CNJ vira vínculo real quando o processo existe.
type intimProcessos struct{ p *processos.Service }

func (a intimProcessos) IDPorNumero(ctx context.Context, numero string) (string, bool, error) {
	id, err := a.p.PorNumero(ctx, numero)
	return id, id != "", err
}

// agendaIntimacoes entrega à Agenda a categoria Intimações.
type agendaIntimacoes struct{ s *intimacoes.Service }

func (a agendaIntimacoes) Lista(ctx context.Context) ([]agenda.IntimacaoLinha, bool, string, error) {
	ls, ok, msg, err := a.s.Lista(ctx)
	out := make([]agenda.IntimacaoLinha, len(ls))
	for i, l := range ls {
		out[i] = agenda.IntimacaoLinha{ID: l.ID, Origem: l.Origem, Processo: l.Processo, Data: l.Data, Status: l.Status,
			StatusRotulo: l.StatusRotulo, StatusClasse: l.StatusClasse, Tarefa: l.Tarefa, TarefaEstado: l.TarefaEstado, Busca: l.Busca}
	}
	return out, ok, msg, err
}

// montarIntimacoes cria o serviço de Intimações. O cliente do DJEN é construído sempre (valida o endereço https), mas só
// é chamado com a coleta ATIVA: desligada, o serviço registra "bloqueado" e não faz nenhuma requisição.
func montarIntimacoes(a *auth.Auth, pool *pgxpool.Pool, proc *processos.Service, tar *tarefas.Service, o *options) (*intimacoes.Service, error) {
	var col intimacoes.Coletor = o.coletor
	if col == nil {
		base := o.djen.BaseURL
		if base == "" {
			base = "https://comunicaapi.pje.jus.br/api/v1/comunicacao"
		}
		c, err := djen.Novo(djen.Config{BaseURL: base, IntervaloMinimo: time.Second, EsperaTaxa: time.Minute, Transport: o.djen.Transport})
		if err != nil {
			return nil, err
		}
		col = c
	}
	var auto intimacoes.TarefaAutomatica = intimTarefas{s: tar}
	if o.envolveTarefas != nil {
		auto = o.envolveTarefas(auto)
	}
	svc := &intimacoes.Service{Pool: pool, Coletas: &coletas.Store{Pool: pool}, Coletor: col, Tarefas: auto,
		TarefasDe: intimTarefas{s: tar}, Processos: intimProcessos{p: proc}, Now: o.agora,
		Cfg: intimacoes.Config{Ativo: o.djen.Ativo, Intervalo: o.djen.Intervalo, AlertaHoras: o.djen.AlertaHoras}}
	a.AddFaixa(func(ctx context.Context, _ *auth.Session) (auth.Faixa, error) {
		forte, texto, ativo, err := svc.Alerta(ctx)
		if err != nil || !ativo {
			return auth.Faixa{}, err
		}
		return auth.Faixa{Strong: forte, Text: texto}, nil
	})
	return svc, nil
}
