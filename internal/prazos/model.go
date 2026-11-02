// Package prazos cuida dos prazos processuais. REGRA DE OURO: um prazo só vale depois de confirmado por advogado.
// O sistema nunca é a única fonte de prazo (ver docs/ROADMAP-PRAZOS-E-INTEGRACOES.md). As datas aqui são dias
// CORRIDOS entre hoje e a data fatal; o cálculo de dias úteis/feriados forenses NÃO é feito (fase própria).
package prazos

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Opcao é valor/rótulo de lista suspensa.
type Opcao = webx.Opcao

// Fontes aceitas.
var Fontes = []Opcao{
	{Valor: "manual", Rotulo: "Digitado por advogado/assistente"}, {Valor: "djen", Rotulo: "DJEN (intimação)"},
	{Valor: "datajud", Rotulo: "DataJud (movimentação)"}, {Valor: "outra", Rotulo: "Outra (PJe, e-mail, telefone…)"},
}

// Situações.
const (
	AConferir  = "a_conferir"
	Confirmado = "confirmado"
	Cumprido   = "cumprido"
	Cancelado  = "cancelado"
)

// ProcessoLookup e UserLookup são os contratos com os outros domínios.
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

// Prazo é a visão de leitura.
type Prazo struct {
	ID, Descricao, Fundamento, Fonte, Origem, Situacao string
	ProcessoID, ProcessoNumero                         string
	ResponsavelID, Responsavel                         string
	ConfirmadoPor                                      string
	DataFatal                                          time.Time
	ConfirmadoEm, CumpridoEm                           *time.Time
	Version                                            int
	DiasRestantes                                      int // negativo = atrasado (dias corridos)
}

// Aberto diz se o prazo ainda corre.
func (p Prazo) Aberto() bool { return p.Situacao == AConferir || p.Situacao == Confirmado }

// Nivel classifica a urgência (só para exibição; usa dias corridos).
func (p Prazo) Nivel() string {
	if !p.Aberto() {
		return "encerrado"
	}
	switch d := p.DiasRestantes; {
	case d < 0:
		return "atrasado"
	case d == 0:
		return "hoje"
	case d <= 3:
		return "d3"
	case d <= 5:
		return "d5"
	case d <= 10:
		return "d10"
	}
	return "normal"
}

// Rotulo descreve o nível em palavras (a cor nunca é a única informação).
func (p Prazo) Rotulo() string {
	switch p.Nivel() {
	case "atrasado":
		return "ATRASADO"
	case "hoje":
		return "VENCE HOJE"
	case "d3":
		return "até 3 dias"
	case "d5":
		return "até 5 dias"
	case "d10":
		return "até 10 dias"
	}
	return ""
}

// Texto diz quantos dias faltam (ou passaram) em dias corridos.
func (p Prazo) Texto() string {
	d := p.DiasRestantes
	switch {
	case !p.Aberto():
		return ""
	case d < 0:
		return pluraliza(-d, "dia de atraso", "dias de atraso")
	case d == 0:
		return "hoje"
	}
	return "faltam " + pluraliza(d, "dia", "dias")
}

func pluraliza(n int, s, p string) string {
	if n == 1 {
		return "1 " + s
	}
	return strconv.Itoa(n) + " " + p
}

// SituacaoRotulo devolve a situação em palavras.
func (p Prazo) SituacaoRotulo() string {
	switch p.Situacao {
	case AConferir:
		return "A CONFERIR PELO ADVOGADO"
	case Confirmado:
		return "Confirmado"
	case Cumprido:
		return "Cumprido"
	}
	return "Cancelado"
}

// Input vem do formulário.
type Input struct{ Descricao, ProcessoID, DataFatal, Fundamento, Fonte, ResponsavelID string }

type normalized struct {
	descricao, processo, fundamento, fonte, responsavel string
	dataFatal                                           time.Time
}

func (in Input) normalize() (normalized, error) {
	n := normalized{descricao: strings.TrimSpace(in.Descricao), processo: strings.TrimSpace(in.ProcessoID),
		fundamento: strings.TrimSpace(in.Fundamento), fonte: in.Fonte, responsavel: strings.TrimSpace(in.ResponsavelID)}
	if c := utf8.RuneCountInString(n.descricao); c < 2 || c > 500 {
		return n, webx.Invalidf("Descreva o prazo (de 2 a 500 caracteres).")
	}
	if utf8.RuneCountInString(n.fundamento) > 300 {
		return n, webx.Invalidf("Fundamento com mais de 300 caracteres.")
	}
	ok := false
	for _, f := range Fontes {
		ok = ok || f.Valor == n.fonte
	}
	if !ok {
		return n, webx.Invalidf("Escolha a fonte do prazo.")
	}
	d, err := time.Parse("2006-01-02", strings.TrimSpace(in.DataFatal))
	if err != nil {
		return n, webx.Invalidf("Informe a data fatal do prazo.")
	}
	n.dataFatal = d
	return n, nil
}
