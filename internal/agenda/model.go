// Package agenda cuida dos compromissos (Reunião, Ligação, Aula) e da página inicial do painel, a Agenda, que reúne por
// categoria também as tarefas, as intimações, as audiências e as diligências (que são de outros domínios, lidos por
// interfaces). É o desenho do painel B: o B vale na forma (rótulos, campos, avisos), o v2 vale na regra (segurança).
package agenda

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa; Ref é a referência curta a uma pessoa.
type (
	Opcao = webx.Opcao
	Ref   = webx.PessoaRef
)

// TipoDiligencia é o tipo do "Novo compromisso" que cria uma diligência (ficha própria, outro domínio).
const TipoDiligencia = "Diligência"

// TiposCompromisso na ordem do B (OPCOES.agendaTipo).
var TiposCompromisso = []string{"Reunião", "Ligação", TipoDiligencia, "Aula"}

func tipoValido(t string) bool {
	for _, x := range TiposCompromisso {
		if x == t {
			return true
		}
	}
	return false
}

// Compromisso é a visão de leitura (data AAAA-MM-DD, hora HH:MM).
type Compromisso struct {
	ID, Tipo, Descricao, Local                        string
	Data                                              time.Time
	Hora                                              string
	ClienteID, AtendimentoID, ProcessoID, CriadoPorID string
	Version                                           int
}

// DataISO preenche o campo <input type="date">.
func (c Compromisso) DataISO() string {
	if c.Data.IsZero() {
		return ""
	}
	return c.Data.Format("2006-01-02")
}

// Input vem do formulário. Pessoas e vínculos são ids; Responsavel só vale para Diligência (texto livre, como no B).
type Input struct {
	Tipo, Descricao, Data, Hora, Local, Responsavel   string
	ClienteID, AtendimentoID, ProcessoID, CriadoPorID string
}

type normalized struct {
	tipo, descricao, local string
	data                   time.Time
	hora                   string
	cliente, atend, proc   string
	criadoPor              string
}

// normalize aplica as validações do B (saveAgenda) pela ordem dele.
func (in Input) normalize() (n normalized, err error) {
	n.tipo = strings.TrimSpace(in.Tipo)
	if !tipoValido(n.tipo) {
		return n, webx.Invalidf("Selecione o tipo do compromisso.")
	}
	n.descricao, n.local = strings.TrimSpace(in.Descricao), strings.TrimSpace(in.Local)
	d, derr := webx.DataISO(in.Data)
	h, hok := webx.HoraHM(in.Hora)
	if n.descricao == "" || derr != nil || !hok {
		return n, webx.Invalidf("Descrição, data e hora são obrigatórios.")
	}
	if utf8.RuneCountInString(n.descricao) > 1000 || utf8.RuneCountInString(n.local) > 1000 {
		return n, webx.Invalidf("Texto longo demais.")
	}
	n.data, n.hora = d, h
	n.cliente, n.atend, n.proc = strings.TrimSpace(in.ClienteID), strings.TrimSpace(in.AtendimentoID), strings.TrimSpace(in.ProcessoID)
	n.criadoPor = strings.TrimSpace(in.CriadoPorID)
	return n, nil
}

// PessoaLookup é o que a Agenda precisa do cadastro de pessoas.
type PessoaLookup interface {
	Refs(ctx context.Context, ids []string) ([]Ref, error) // ids nulo = todas as pessoas não excluídas
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
}

// VinculoLookup é o que a Agenda precisa de atendimentos e processos (o adaptador mora em internal/app).
type VinculoLookup interface {
	// Opcoes monta o seletor do tipo ("atendimento" | "processo") com os rótulos do B.
	Opcoes(ctx context.Context, tipo string) ([]Opcao, error)
	// Resumos devolve id → código ("ATE. 0000001" / "PRO. 0000001").
	Resumos(ctx context.Context, tipo string, ids []string) (map[string]string, error)
	Existe(ctx context.Context, tipo, id string) (bool, error)
}

// Conflito descreve uma audiência que ocupa o mesmo dia e a mesma hora (Data em AAAA-MM-DD, como o B imprime).
type Conflito struct{ Titulo, Data, Hora string }

// AudienciaLookup é o que a Agenda precisa do domínio Audiências.
type AudienciaLookup interface {
	EmAberto(ctx context.Context, hoje time.Time) ([]AudienciaLinha, error)
	Total(ctx context.Context) (int, error)
	// NoHorario devolve as audiências (não canceladas) do mesmo dia e da mesma hora exata.
	NoHorario(ctx context.Context, data time.Time, hora string) ([]Conflito, error)
}

// DiligenciaNova é o que o "Novo compromisso" do tipo Diligência entrega ao domínio Diligências.
type DiligenciaNova struct{ Titulo, ProcessoID, ClienteID, Local, DataLimite, Hora, Responsavel string }

// DiligenciaLookup é o que a Agenda precisa do domínio Diligências.
type DiligenciaLookup interface {
	EmAberto(ctx context.Context, hoje time.Time) ([]DiligenciaLinha, error)
	Total(ctx context.Context) (int, error)
	Criar(ctx context.Context, ac webx.Actor, n DiligenciaNova) (codigo string, err error)
}

// TarefaLookup é o que a Agenda precisa do domínio Tarefas.
type TarefaLookup interface {
	// EmAberto devolve as tarefas fora dos estados concluída e cancelada (categoria Tarefas).
	EmAberto(ctx context.Context, hoje time.Time) ([]TarefaLinha, error)
	// PorSubtitulo devolve as em aberto cujo subtítulo, sem acento e em minúsculas, é um dos normalizados.
	PorSubtitulo(ctx context.Context, hoje time.Time, normalizados []string) ([]TarefaLinha, error)
	// TitulosComPrazoEm devolve os títulos das tarefas em aberto com prazo de entrega nesse dia (aviso de mesmo dia).
	TitulosComPrazoEm(ctx context.Context, data time.Time) ([]string, error)
}

// IntimacaoLookup é o que a Agenda precisa do domínio Intimações (Plano 5). Até lá: disponivel=false.
type IntimacaoLookup interface {
	Lista(ctx context.Context) (linhas []IntimacaoLinha, disponivel bool, mensagem string, err error)
}

// AudienciaLinha é uma audiência na tabela da Agenda (datas já em dd/mm/aaaa).
type AudienciaLinha struct {
	ID, TipoRotulo, Codigo, Titulo, Data, Hora, Local, Processo string
	Cancelada                                                   bool
	Busca                                                       string
}

// DiligenciaLinha é uma diligência na tabela da Agenda.
type DiligenciaLinha struct {
	ID, Codigo, Titulo, DataLimite, Hora, Local, Responsavel, Status, Processo string
	Busca                                                                      string
}

// TarefaLinha é uma tarefa na tabela da Agenda (as colunas e os selos de tabelaTarefasCategoriaAgenda do B).
type TarefaLinha struct {
	Codigo, Titulo, CriadoPor, Prazo, PrazoFatal string
	Clientes                                     []Ref
	StatusRotulo, StatusClasse                   string
	FatalAConferir                               bool // regra do v2: o selo A CONFERIR vai no mesmo campo do prazo fatal
	Busca                                        string
}

// IntimacaoLinha é uma intimação na tabela da Agenda (Plano 5).
type IntimacaoLinha struct {
	ID, Origem, Processo, Data, Status, StatusRotulo, StatusClasse, Tarefa, TarefaEstado string
	Busca                                                                                string
}
