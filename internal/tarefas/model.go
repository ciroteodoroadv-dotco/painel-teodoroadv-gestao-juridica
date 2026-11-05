// Package tarefas cuida das tarefas do escritório, no desenho do painel B: toda tarefa nasce vinculada a um atendimento
// ou a um processo, tem 7 estados, prioridade, partes e responsáveis (pessoas) e prazo fatal. O prazo fatal é SEMPRE
// informado por gente: o sistema não calcula prazo, e o domínio Prazos guarda a data como "a conferir" até um advogado confirmar.
package tarefas

import (
	"context"
	"io"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa; Ref é a referência curta a uma pessoa.
type (
	Opcao = webx.Opcao
	Ref   = webx.PessoaRef
)

// ParteIn é uma linha de "Partes" do formulário; ParteTarefa é a mesma linha já lida do banco.
type (
	ParteIn     struct{ PessoaID, Papel string }
	ParteTarefa struct{ PessoaID, Nome, Papel string }
)

// Tarefa é a visão de leitura.
type Tarefa struct {
	ID, Codigo, Titulo, Subtitulo, Estado, Prioridade string
	OrigemTipo, OrigemID, OrigemCodigo, ClienteNome   string
	IntimacaoID                                       string // intimação (DJEN) de onde a tarefa nasceu, se nasceu
	Descricao, Observacoes                            string
	TarefaPaiID, TarefaPaiCodigo                      string
	DataBase, Prazo, PrazoFatal                       *time.Time
	PrazoFatalSituacao                                string // ""|"a_conferir"|"confirmado"|"cumprido"|"cancelado"
	CriadoPorID, CriadoPor                            string
	Partes                                            []ParteTarefa
	Responsaveis                                      []Ref
	Version                                           int
	PrazoInterno                                      bool   // prazo fatal interno de confirmação (2 dias), não o prazo processual
	Fonte                                             string // DJEN, DataJud ou DJEN (sigiloso), nas tarefas de análise
}

func br(t *time.Time) string {
	if t == nil {
		return "—"
	}
	return t.Format("02/01/2006")
}

func iso(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// PrazoBR, PrazoFatalBR e DataBaseBR são as datas no formato dd/mm/aaaa (ou "—").
func (t Tarefa) PrazoBR() string      { return br(t.Prazo) }
func (t Tarefa) PrazoFatalBR() string { return br(t.PrazoFatal) }
func (t Tarefa) DataBaseBR() string   { return br(t.DataBase) }

// PrazoISO, PrazoFatalISO e DataBaseISO preenchem os campos <input type="date">.
func (t Tarefa) PrazoISO() string      { return iso(t.Prazo) }
func (t Tarefa) PrazoFatalISO() string { return iso(t.PrazoFatal) }
func (t Tarefa) DataBaseISO() string   { return iso(t.DataBase) }

// AConferir diz se o prazo fatal ainda espera confirmação de advogado (selo A CONFERIR no mesmo campo).
func (t Tarefa) AConferir() bool { return t.PrazoFatalSituacao == "a_conferir" }

// Aberta diz se a tarefa ainda corre (não está concluída nem cancelada).
func (t Tarefa) Aberta() bool { return Aberto(t.Estado) }

// ResponsaveisTexto lista os responsáveis separados por vírgula (a célula "Responsável" do B).
func (t Tarefa) ResponsaveisTexto() string {
	s := ""
	for i, r := range t.Responsaveis {
		if i > 0 {
			s += ", "
		}
		s += r.Nome
	}
	return s
}

// Input vem do formulário. Datas em AAAA-MM-DD; pessoas e vínculos são ids.
type Input struct {
	OrigemTipo, OrigemID, TarefaPai, Titulo, Subtitulo, Descricao string
	CriadoPorID, DataBase, Prazo, PrazoFatal                      string
	Estado, Prioridade, Observacoes                               string
	Partes                                                        []ParteIn
	Responsaveis                                                  []string
}

// PessoaLookup é o que este domínio precisa do cadastro de pessoas.
type PessoaLookup interface {
	Refs(ctx context.Context, ids []string) ([]Ref, error)
	Opcoes(ctx context.Context, ac webx.Actor, comDocumento bool) ([]Opcao, error)
}

// VinculoLookup é o que este domínio precisa de atendimentos e processos (o adaptador mora em internal/app).
type VinculoLookup interface {
	// Resumos devolve id → {código, nome do cliente}; id inexistente fica de fora.
	Resumos(ctx context.Context, tipo string, ids []string) (map[string][2]string, error)
	// Opcoes monta o seletor "Vínculo específico": "ATE. 0000001 · CLIENTE" / "PRO. 0000012 · CLIENTE".
	Opcoes(ctx context.Context, tipo string) ([]Opcao, error)
	// Clientes devolve id → clientes (nome e código PES) do atendimento/processo, para a tabela de tarefas da Agenda.
	Clientes(ctx context.Context, tipo string, ids []string) (map[string][]Ref, error)
}

// PrazoSync espelha o prazo fatal da tarefa no domínio Prazos (o adaptador mora em internal/app). As escritas acontecem
// DENTRO da transação da tarefa: ou tarefa e prazo são gravados juntos, ou nenhum dos dois (auditoria 05/11, item 1.3).
// Nenhuma escrita daqui confirma prazo: o prazo da tarefa nasce e permanece "a conferir" até o ato explícito Confirmar.
type PrazoSync interface {
	// SincronizarTx cria ou atualiza o prazo da tarefa. Devolve a situação resultante.
	SincronizarTx(ctx context.Context, tx pgx.Tx, ac webx.Actor, tarefaID, processoID, descricao string, data *time.Time) (situacao string, err error)
	// EncerrarTx fecha o prazo da tarefa: motivo "cumprida", "cancelada" ou "excluida".
	EncerrarTx(ctx context.Context, tx pgx.Tx, ac webx.Actor, tarefaID, motivo string) error
	// Situacoes devolve tarefa → situação do prazo (só as que têm prazo).
	Situacoes(ctx context.Context, tarefaIDs []string) (map[string]string, error)
	// Confirmar usa o fluxo de confirmação do domínio Prazos (só advogado; a rota exige a permissão).
	Confirmar(ctx context.Context, ac webx.Actor, tarefaID string) error
}

// DocumentoSink guarda o PDF enviado pela tarefa no domínio Documentos (cifrado, conferido pelo conteúdo, auditado).
type DocumentoSink interface {
	Anexar(ctx context.Context, ac webx.Actor, tarefaID, origemTipo, origemID, nome string, conteudo io.Reader) error
}
