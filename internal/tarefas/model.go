// Package tarefas cuida das tarefas do escritório. O vencimento é SEMPRE informado pelo advogado:
// o sistema não calcula prazo processual nem decide providência.
package tarefas

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Status possíveis.
var Statuses = []Opcao{
	{Valor: "aberta", Rotulo: "Aberta"}, {Valor: "em_andamento", Rotulo: "Em andamento"}, {Valor: "em_revisao", Rotulo: "Em revisão"},
	{Valor: "concluida", Rotulo: "Concluída"}, {Valor: "cancelada", Rotulo: "Cancelada"},
}

// Prioridades possíveis (ordem da lista suspensa).
var Prioridades = []Opcao{
	{Valor: "baixa", Rotulo: "Baixa"}, {Valor: "normal", Rotulo: "Normal"},
	{Valor: "media", Rotulo: "Média"}, {Valor: "alta", Rotulo: "Alta"},
}

func rotulo(v string) string { return rotuloDe(Statuses, v) }

func rotuloDe(lista []Opcao, v string) string {
	for _, o := range lista {
		if o.Valor == v {
			return o.Rotulo
		}
	}
	return v
}

// ProcessoLookup é o que este domínio precisa do domínio de processos (interface explícita).
type ProcessoLookup interface {
	Existe(ctx context.Context, id string) (bool, error)
	Resumos(ctx context.Context, ids []string) (map[string]string, error)
	Buscar(ctx context.Context, q string, limit int) ([]Opcao, error)
}

// UserLookup é o que este domínio precisa do cadastro de usuários.
type UserLookup interface {
	Options(ctx context.Context) ([]Opcao, error)
	Emails(ctx context.Context, ids []string) (map[string]string, error)
}

// Tarefa é a visão de leitura.
type Tarefa struct {
	ID, Titulo, Status, Origem string
	ProcessoID, ProcessoNumero string
	ResponsavelID, Responsavel string
	Vencimento                 *time.Time
	Version                    int
	Subtitulo, Prioridade      string
	Atrasada                   bool // só exibição: vencimento anterior a hoje e ainda em aberto
}

// StatusRotulo devolve o rótulo do status.
func (t Tarefa) StatusRotulo() string { return rotulo(t.Status) }

// Aberta diz se a tarefa ainda está pendente.
func (t Tarefa) Aberta() bool {
	return t.Status == "aberta" || t.Status == "em_andamento" || t.Status == "em_revisao"
}

// AConferir indica tarefa criada por automação, pendente de conferência pelo advogado.
func (t Tarefa) AConferir() bool { return t.Origem == "automacao_a_conferir" }

// Input vem do formulário.
type Input struct {
	Titulo, ProcessoID, ResponsavelID, Vencimento, Status string
	Subtitulo, Prioridade                                 string
}

type normalized struct {
	titulo, processo, responsavel, status string
	subtitulo, prioridade                 string
	vencimento                            *time.Time
}

func (in Input) normalize() (normalized, error) {
	n := normalized{titulo: strings.TrimSpace(in.Titulo), processo: strings.TrimSpace(in.ProcessoID),
		responsavel: strings.TrimSpace(in.ResponsavelID), status: in.Status}
	if c := utf8.RuneCountInString(n.titulo); c < 2 || c > 300 {
		return n, webx.Invalidf("Informe o título da tarefa (de 2 a 300 caracteres).")
	}
	if rotulo(n.status) == n.status {
		return n, webx.Invalidf("Escolha o status.")
	}
	n.subtitulo = strings.TrimSpace(in.Subtitulo)
	if utf8.RuneCountInString(n.subtitulo) > 80 {
		return n, webx.Invalidf("O subtítulo pode ter no máximo 80 caracteres.")
	}
	n.prioridade = strings.TrimSpace(in.Prioridade)
	if n.prioridade == "" {
		n.prioridade = "normal"
	}
	if rotuloDe(Prioridades, n.prioridade) == n.prioridade {
		return n, webx.Invalidf("Escolha a prioridade.")
	}
	if v := strings.TrimSpace(in.Vencimento); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return n, webx.Invalidf("Data de vencimento inválida.")
		}
		n.vencimento = &t
	}
	return n, nil
}

// Hoje devolve a data de hoje no fuso do escritório (America/Sao_Paulo), à meia-noite UTC para comparar com DATE.
func Hoje(now time.Time) time.Time {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.FixedZone("BRT", -3*3600)
	}
	n := now.In(loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

// PrioridadeRotulo devolve o rótulo da prioridade (vazio conta como normal).
func (t Tarefa) PrioridadeRotulo() string {
	if t.Prioridade == "" {
		return "Normal"
	}
	return rotuloDe(Prioridades, t.Prioridade)
}

// PrioridadeClasse devolve a classe CSS do selo de prioridade.
func (t Tarefa) PrioridadeClasse() string {
	if t.Prioridade == "" {
		return "prioridade-normal"
	}
	return "prioridade-" + t.Prioridade
}
