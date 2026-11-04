// Package intimacoes recebe as intimações do DJEN pelas OABs cadastradas e as mostra na categoria Intimações da Agenda.
// O sistema DETECTA e SUGERE; quem confirma é o advogado. Cada intimação nova abre uma tarefa de análise em "A confirmar",
// SEM prazo: o sistema nunca presume prazo, e "nenhuma intimação encontrada" nunca significa "nenhum prazo correndo".
// A integração vem desligada por padrão (PAINEL_DJEN_ATIVO=false) e, desligada, não faz nenhuma requisição de rede.
package intimacoes

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/intimacoes/djen"
	"painelteodoro/internal/platform/webx"
)

// Modulo é o nome da coleta no rodapé "Estado das coletas" (o B chama a mesma coleta de "intimacoes", rótulo DJEN).
const Modulo = "intimacoes"

// BaseAutorizacaoPadrao é o texto proposto no cadastro da OAB (determinação do advogado titular do escritório).
const BaseAutorizacaoPadrao = "Distribuição dos processos em nome do advogado — responsabilidade do titular do escritório"

// Config da coleta.
type Config struct {
	Ativo       bool
	Intervalo   time.Duration
	AlertaHoras int
}

// Coletor é o cliente do DJEN (djen.Cliente); nos testes, um falso.
type Coletor interface {
	Coletar(ctx context.Context, oab, uf string, desde, ate time.Time) (djen.Resultado, error)
}

// AnaliseIn é o pedido da tarefa de análise (a tabela de tarefas é de outro domínio; o adaptador em internal/app liga os dois).
type AnaliseIn struct {
	IntimacaoID, ProcessoID, Titulo, Descricao string
	CriadoPor, AuditActor, RequestID           string
	PrazoFatal                                 *time.Time // prazo INTERNO de confirmação (E3 do Plano 6)
}

// TarefaAutomatica cria a tarefa de análise dentro da transação da intimação. Idempotente por intimação.
type TarefaAutomatica interface {
	CriarAnaliseDeIntimacao(ctx context.Context, tx pgx.Tx, in AnaliseIn) (tarefaID string, err error)
	// CriarConsolidadaDoDia abre (ou completa) a tarefa consolidada do processo no dia, também na transação do chamador.
	CriarConsolidadaDoDia(ctx context.Context, tx pgx.Tx, in ConsolidadaIn) (tarefaID string, err error)
	// EncerrarPrazoInterno encerra o prazo interno da tarefa de análise quando a intimação é decidida.
	EncerrarPrazoInterno(ctx context.Context, ac webx.Actor, intimacaoID string) error
}

// TarefaResumo é a tarefa de análise como a tabela a mostra: título e rótulo do estado.
type TarefaResumo struct{ Titulo, Estado string }

// TarefasDeIntimacao lê as tarefas de análise (por id de intimação).
type TarefasDeIntimacao interface {
	PorIntimacoes(ctx context.Context, ids []string) (map[string]TarefaResumo, error)
}

// ProcessoPorNumero resolve o número CNJ (com máscara) no processo do v2, se existir.
type ProcessoPorNumero interface {
	IDPorNumero(ctx context.Context, numero string) (id string, existe bool, err error)
}

// OAB é uma OAB cadastrada para a coleta. A autorização tem base (texto) e data; sem os dois a coleta é recusada.
type OAB struct {
	ID, Nome, Numero, UF, AutorizacaoBase string
	AutorizacaoEm                         time.Time
	Ativa                                 bool
	UltimoSucesso                         *time.Time
	CriadoPor                             string
	Version                               int
}

// AutorizacaoISO preenche o campo de data do formulário.
func (o OAB) AutorizacaoISO() string { return o.AutorizacaoEm.Format("2006-01-02") }

// Rotulo é "123456/SP".
func (o OAB) Rotulo() string { return o.Numero + "/" + o.UF }

// OABIn vem do formulário "Nova OAB" (data AAAA-MM-DD).
type OABIn struct {
	Nome, Numero, UF, AutorizacaoBase, AutorizacaoEm string
	Ativa                                            bool
}

// Linha é uma intimação na tabela da Agenda.
type Linha struct {
	ID, Origem, Processo, Data, Status, StatusRotulo, StatusClasse string
	Tarefa, TarefaEstado, Busca                                    string
}

// Resumo é o resultado de uma rodada de coleta (só contagens; nunca texto de intimação).
type Resumo struct {
	Status, Mensagem                              string
	OABs, Novas, Repetidas, FalhasItem, FalhasOAB int
	Recusadas                                     int
}

// Rotulos e classes do B (statusIntimacaoLabel / statusIntimacaoClasse).
func statusRotulo(s string) string {
	switch s {
	case "confirmada":
		return "Confirmada"
	case "cancelada":
		return "Cancelada"
	}
	return "Em triagem"
}

func statusClasse(s string) string {
	switch s {
	case "confirmada":
		return "badge-ok"
	case "cancelada":
		return "badge-neutral"
	}
	return "badge-warn"
}
