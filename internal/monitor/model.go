// Package monitor acompanha, processo a processo, as movimentações dos processos cadastrados no DataJud (CNJ) e, para os
// processos em segredo de justiça, as comunicações do DJEN filtradas pelo NÚMERO do processo. Cada movimentação nova vira uma
// linha na categoria Intimações da Agenda e uma tarefa de análise em "A confirmar". O sistema DETECTA; quem confirma é o
// advogado. O DataJud não traz o teor da intimação e o sistema não calcula nem presume prazo processual.
// A integração vem desligada por padrão (PAINEL_DATAJUD_ATIVO=false) e, desligada, não faz nenhuma requisição de rede.
package monitor

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/monitor/datajud"
	"painelteodoro/internal/platform/coletas"
)

// Módulos no rodapé "Estado das coletas" (rótulos do B: DataJud e DJEN por processo sigiloso).
const (
	Modulo          = "monitor"
	ModuloSigilosos = "monitor-segredos"
)

// Textos fixos dos estados.
const (
	MsgDesligada   = "Coleta desligada neste ambiente."
	MsgSemChave    = datajud.MsgSemChave
	MsgSemRegistro = "Sem registro no DataJud para este número (pode haver segredo de justiça, atraso na publicação ou número incorreto). Confira nas fontes oficiais."
	MsgSemNumero   = "Processo sem número CNJ válido: não é consultado. Complete o número na ficha do processo."
	MsgSigilo      = "Processo em segredo de justiça no DataJud: as movimentações não são devolvidas. Cadastre-o em \"Processos em segredo de justiça\" para consulta pelo DJEN."
)

// Config da coleta.
type Config struct {
	Ativo       bool
	Intervalo   time.Duration
	AlertaHoras int
}

// ProcessoRef é o processo cadastrado como o monitor o enxerga (implementado por Processos, via adaptador).
type ProcessoRef struct{ ID, Numero, CriadoPor string }

// Processos lista os processos a monitorar: todos os não excluídos, em qualquer estado (E1 do Plano 6).
type Processos interface {
	Monitoraveis(ctx context.Context) ([]ProcessoRef, error)
	Ref(ctx context.Context, id string) (ProcessoRef, bool, error)
}

// OABRef é a OAB cadastrada (domínio Intimações).
type OABRef struct {
	ID, Numero, UF, AutorizacaoBase string
	AutorizacaoEm                   time.Time
	Ativa                           bool
}

// OABs lê uma OAB cadastrada.
type OABs interface {
	Obter(ctx context.Context, id string) (OABRef, error)
	Listar(ctx context.Context) ([]OABRef, error)
}

// Consultor é o cliente do DataJud (datajud.Cliente); nos testes, o mesmo cliente apontado para um httptest.Server.
type Consultor interface {
	Consultar(ctx context.Context, alias, numeroCNJ string) (datajud.Resultado, error)
	TemChave() bool
}

// NovaIntimacao é o pedido de linha em Intimações (o adaptador liga ao domínio Intimações).
type NovaIntimacao struct {
	Hash, Origem, OabID, Tribunal, Orgao, ProcessoNumero, ProcessoID, Tipo string
	Data                                                                   *time.Time
	Texto, Resumo, Link, TituloAnalise, DescricaoAnalise, LinhaConsolidada string
	CriadoPor, Ator, RequestID                                             string
}

// Registro grava a intimação, a tarefa de análise e a consolidada do dia na transação do chamador.
type Registro interface {
	RegistrarNaTx(ctx context.Context, tx pgx.Tx, n NovaIntimacao) (id string, repetida bool, err error)
}

// ComunicacaoSigilosa é uma comunicação do DJEN do processo em segredo de justiça.
type ComunicacaoSigilosa struct {
	Hash, Tribunal, Orgao, Processo, Tipo, Texto, Resumo, Link string
	Data                                                       time.Time
}

// Sigilosos consulta o DJEN pela OAB FILTRADA pelo número do processo (nunca a pauta inteira).
type Sigilosos interface {
	ColetarPorNumero(ctx context.Context, oab, uf, numero string, desde, ate time.Time) ([]ComunicacaoSigilosa, error)
}

// Service concentra o monitor. Só toca nas próprias tabelas; os outros domínios entram por interfaces.
type Service struct {
	Pool      *pgxpool.Pool
	Coletas   *coletas.Store
	Cliente   Consultor
	Processos Processos
	OABs      OABs
	Registro  Registro
	Sigilosos Sigilosos
	Cfg       Config
	Now       func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Resumo de uma rodada (só contagens; nunca texto de movimentação).
type Resumo struct {
	Status, Mensagem                                                 string
	Processos, Sucesso, SemRegistro, SemNumero, Falhas, Novas, Bases int
}
