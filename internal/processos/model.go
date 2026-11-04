// Package processos cuida do cadastro de processos, suas partes, andamentos e financeiro, nos campos do painel B.
// O número é validado pelo dígito verificador (Res. CNJ 65). O sistema NÃO calcula prazo.
package processos

import (
	"context"
	"strconv"
	"strings"
	"time"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Estado é um grupo do acordeão (CATEGORIAS_PROCESSOS do B).
type Estado struct{ Chave, Valor, Rotulo string }

// Estados na ordem do B. Valor é o que se grava; Rotulo é o título do grupo.
var Estados = []Estado{
	{"ativos", "ativo", "Ativos"}, {"sobrestados", "sobrestado", "Sobrestados"},
	{"extintos", "extinto", "Extintos"}, {"arquivados", "arquivado", "Arquivados definitivamente"},
}

func estadoValido(v string) bool {
	for _, e := range Estados {
		if e.Valor == v {
			return true
		}
	}
	return false
}

// Listas de seleção copiadas literalmente do painel B (05-dominio.js, 06-helpers-vinculos.js, 09-processos.js).
var (
	Sistemas          = []string{"PJe", "Eproc", "e-SAJ", "Projudi", "Físico", "Outro"}
	Instancias        = []string{"1º grau", "2º grau", "Tribunais Superiores", "Juizado Especial", "Outro"}
	PapeisProcessuais = []string{"Autor", "Réu", "Exequente", "Executado", "Embargante", "Terceiro interessado", "Assistente", "Outro"}
	Classes           = []string{"Procedimento Comum Cível", "Execução Fiscal", "Execução de Título Extrajudicial", "Embargos à Execução Fiscal", "Embargos de Terceiro", "Mandado de Segurança", "Ação Anulatória de Débito Fiscal", "Ação Declaratória", "Ação de Repetição de Indébito", "Cumprimento de Sentença", "Medida Cautelar Fiscal", "Exceção de Pré-Executividade", "Agravo de Instrumento", "Apelação Cível", "Recurso Especial", "Recurso Extraordinário", "Outra (especificar)"}
	Materias          = []string{"Tributário", "Cível / Processual Civil", "Bancário", "Consumidor", "Empresarial", "Previdenciário", "Trabalhista", "Administrativo", "Ambiental", "Sucessões e Família", "Outra"}
)

// PessoaLookup é o que este domínio precisa do domínio de pessoas (interface explícita, sem SQL cruzado).
type PessoaLookup interface {
	Names(ctx context.Context, ids []string) (map[string]string, error)
	Exist(ctx context.Context, ids []string) (map[string]bool, error)
	Opcoes(ctx context.Context, ac webx.Actor, comDocumento bool) ([]pessoas.Opcao, error)
}

// AtendimentoLookup é o que este domínio precisa do domínio de atendimentos (seletor "Atendimento de origem").
type AtendimentoLookup interface {
	OpcoesOrigem(ctx context.Context) ([]Opcao, error)
	Existe(ctx context.Context, id string) (bool, error)
}

// VinculosLookup é implementado por quem tem trabalho em aberto ligado ao processo (tarefas, prazos).
type VinculosLookup interface {
	AbertosDoProcesso(ctx context.Context, processoID string) (int, error)
}

// Parte liga uma pessoa ao processo, com o papel processual e a marca "é cliente do escritório".
type Parte struct {
	PessoaID, Nome, Papel string
	Cliente               bool
}

// Processo é a visão de leitura (datas em AAAA-MM-DD ou vazias).
type Processo struct {
	ID, Codigo, Titulo, Numero, Estado, Polo              string
	Sistema, Instancia, Classe, Materia, Comarca, Juizo   string
	Contingencia, Valor                                   string
	Distribuicao, Citacao, Audiencia, Sentenca, Recursos  string
	FaseExecucao, AtendimentoOrigem, Projeto, Observacoes string
	ProxOQue, ProxAte, ProxQuem                           string
	DuplicadoJustificativa, DuplicadoDe                   string
	Duplicado                                             bool
	Partes                                                []Parte
	Responsaveis                                          []string // ids de pessoa, na ordem
	Version                                               int
	CriadoEm, AtualizadoEm                                time.Time
}

// ClienteNome é o nome da primeira parte marcada como cliente.
func (p Processo) ClienteNome() string {
	for _, pt := range p.Partes {
		if pt.Cliente {
			return pt.Nome
		}
	}
	return ""
}

// ParteContraria lista os nomes das demais partes (as que não são a primeira cliente).
func (p Processo) ParteContraria() []string {
	var out []string
	pulou := false
	for _, pt := range p.Partes {
		if pt.Cliente && !pulou {
			pulou = true
			continue
		}
		out = append(out, pt.Nome)
	}
	return out
}

// NumeroOuTitulo é o texto do cabeçalho do drawer de edição (numeroProcesso || title).
func (p Processo) NumeroOuTitulo() string {
	if p.Numero != "" {
		return p.Numero
	}
	return p.Titulo
}

// ValorBR devolve o valor da causa como "R$ 10.000,00" ou vazio.
func (p Processo) ValorBR() string { return formatBRL(p.Valor) }

// DataBR devolve a data AAAA-MM-DD como 03/11/2026 (fmtDate do B); vazia vira "—".
func DataBR(iso string) string {
	t, err := time.Parse("2006-01-02", iso)
	if err != nil {
		return "—"
	}
	return t.Format("02/01/2006")
}

// DistribuicaoBR é a coluna Distribuição da lista.
func (p Processo) DistribuicaoBR() string { return DataBR(p.Distribuicao) }

// formatBRL converte "10000.50" (numeric) em "R$ 10.000,50".
func formatBRL(numeric string) string {
	if numeric == "" {
		return ""
	}
	inteiro, cent, _ := strings.Cut(numeric, ".")
	cent = (cent + "00")[:2]
	var b strings.Builder
	for i, c := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return "R$ " + b.String() + "," + cent
}

// numericDeBRL converte "R$ 10.000,50" (ou só dígitos de centavos) em "10000.50"; vazio devolve "".
func numericDeBRL(v string) (string, bool) {
	var d strings.Builder
	for _, r := range v {
		if r >= '0' && r <= '9' {
			d.WriteRune(r)
		}
	}
	s := strings.TrimLeft(d.String(), "0")
	if d.Len() == 0 {
		return "", true
	}
	if len(s) > 15 {
		return "", false
	}
	for len(s) < 3 {
		s = "0" + s
	}
	return s[:len(s)-2] + "." + s[len(s)-2:], true
}

// LinhaPendencias é o conteúdo do badge de pendências.
func (p Processo) LinhaPendencias() string { return strings.Join(Pendencias(p), "; ") }

// NumPendencias é o número mostrado no badge.
func (p Processo) NumPendencias() string { return strconv.Itoa(len(Pendencias(p))) }

// Contexto de quem lê: define se o documento aparece inteiro nos seletores de pessoa.
func ComDocumento(s *auth.Session) bool { return auth.Can(s.Role, auth.PermPessoasWrite) }
