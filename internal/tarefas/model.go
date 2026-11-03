// Package tarefas cuida das tarefas do escritório, no desenho do painel B: toda tarefa nasce vinculada a um atendimento
// ou a um processo, tem 7 estados, prioridade, partes e responsáveis (pessoas) e prazo fatal. O prazo fatal é SEMPRE
// informado por gente: o sistema não calcula prazo, e o domínio Prazos guarda a data como "a conferir" até um advogado confirmar.
package tarefas

import (
	"context"
	"io"
	"time"

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
	Descricao, Observacoes                            string
	TarefaPaiID, TarefaPaiCodigo                      string
	DataBase, Prazo, PrazoFatal                       *time.Time
	PrazoFatalSituacao                                string // ""|"a_conferir"|"confirmado"|"cumprido"|"cancelado"
	CriadoPorID, CriadoPor                            string
	Partes                                            []ParteTarefa
	Responsaveis                                      []Ref
	Version                                           int
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

// PrazoSync espelha o prazo fatal da tarefa no domínio Prazos (o adaptador mora em internal/app).
type PrazoSync interface {
	// Sincronizar cria ou atualiza o prazo da tarefa. confirmar=true (ator com permissão de advogado) grava como
	// confirmado; senão fica "a conferir". Devolve a situação resultante.
	Sincronizar(ctx context.Context, ac webx.Actor, tarefaID, processoID, descricao string, data *time.Time, confirmar bool) (situacao string, err error)
	// Situacoes devolve tarefa → situação do prazo (só as que têm prazo).
	Situacoes(ctx context.Context, tarefaIDs []string) (map[string]string, error)
	// Encerrar fecha o prazo da tarefa: motivo "cumprida", "cancelada" ou "excluida".
	Encerrar(ctx context.Context, ac webx.Actor, tarefaID, motivo string) error
	// Confirmar usa o fluxo de confirmação do domínio Prazos (só advogado; a rota exige a permissão).
	Confirmar(ctx context.Context, ac webx.Actor, tarefaID string) error
}

// DocumentoSink guarda o PDF enviado pela tarefa no domínio Documentos (cifrado, conferido pelo conteúdo, auditado).
type DocumentoSink interface {
	Anexar(ctx context.Context, ac webx.Actor, tarefaID, origemTipo, origemID, nome string, conteudo io.Reader) error
}

// Reauth confere a senha do usuário logado antes de uma ação destrutiva.
type Reauth interface {
	Confirmar(ctx context.Context, ac webx.Actor, senha string) error
}
