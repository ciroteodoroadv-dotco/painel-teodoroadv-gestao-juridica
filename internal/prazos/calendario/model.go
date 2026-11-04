// Package calendario é o cadastro do calendário forense: feriados, suspensões e recessos de cada tribunal,
// cada dia com a FONTE citada. Sem fonte não grava; nada se apaga (dia errado é inativado com motivo).
package calendario

import (
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/platform/webx"
)

// Tipos de dia aceitos.
var Tipos = []webx.Opcao{
	{Valor: "feriado", Rotulo: "Feriado"},
	{Valor: "suspensao_prazo", Rotulo: "Suspensão de prazo (portaria)"},
	{Valor: "recesso", Rotulo: "Recesso"},
	{Valor: "sem_expediente", Rotulo: "Sem expediente"},
	{Valor: "ponto_facultativo", Rotulo: "Ponto facultativo"},
}

// Ambitos aceitos.
var Ambitos = []webx.Opcao{
	{Valor: "nacional", Rotulo: "Nacional"}, {Valor: "federal", Rotulo: "Justiça Federal"},
	{Valor: "estadual", Rotulo: "Justiça Estadual"}, {Valor: "trabalhista", Rotulo: "Justiça do Trabalho"},
	{Valor: "eleitoral", Rotulo: "Justiça Eleitoral"}, {Valor: "outro", Rotulo: "Outro"},
}

// Calendario é um calendário cadastrado. Verificar=true mostra o selo [VERIFICAR] (o advogado ainda não validou).
type Calendario struct {
	ID, Sigla, Nome, Ambito string
	Verificar               bool
}

// Dia é a visão de leitura de um dia do calendário.
type Dia struct {
	ID, CalendarioID, Tipo, Descricao, FonteCitacao, FonteURL, Observacao string
	VerificadoPor, MotivoInativacao                                       string
	Data                                                                  time.Time
	ContaNaoUtil, Ativo                                                   bool
}

// Verificar diz se o dia ainda carece de verificação humana.
func (d Dia) Verificar() bool { return d.VerificadoPor == "" }

// CalInput é o formulário "Novo calendário".
type CalInput struct{ Sigla, Nome, Ambito string }

// DiaInput é o formulário "Novo dia de calendário".
type DiaInput struct {
	CalendarioID, Data, Tipo, Descricao, FonteCitacao, FonteURL, Observacao string
	ContaNaoUtil                                                            bool
}

func tamanho(s string, min, max int) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	return n >= min && n <= max
}

func valorIn(v string, opts []webx.Opcao) bool {
	for _, o := range opts {
		if o.Valor == v {
			return true
		}
	}
	return false
}

func (in CalInput) validar() (CalInput, error) {
	in.Sigla, in.Nome = strings.ToUpper(strings.TrimSpace(in.Sigla)), strings.TrimSpace(in.Nome)
	switch {
	case !tamanho(in.Sigla, 2, 30):
		return in, webx.Invalidf("Informe a sigla do calendário (2 a 30 caracteres).")
	case !tamanho(in.Nome, 2, 120):
		return in, webx.Invalidf("Informe o nome do calendário.")
	case !valorIn(in.Ambito, Ambitos):
		return in, webx.Invalidf("Âmbito inválido.")
	}
	return in, nil
}

func (in DiaInput) validar() (DiaInput, time.Time, error) {
	in.Descricao, in.FonteCitacao = strings.TrimSpace(in.Descricao), strings.TrimSpace(in.FonteCitacao)
	in.FonteURL, in.Observacao = strings.TrimSpace(in.FonteURL), strings.TrimSpace(in.Observacao)
	data, err := webx.DataISO(in.Data)
	switch {
	case !webx.UUID(in.CalendarioID):
		return in, data, webx.Invalidf("Escolha o calendário.")
	case err != nil || data.Year() < 2000 || data.Year() > 2100:
		return in, data, webx.Invalidf("Data inválida.")
	case !valorIn(in.Tipo, Tipos):
		return in, data, webx.Invalidf("Tipo inválido.")
	case !tamanho(in.Descricao, 2, 200):
		return in, data, webx.Invalidf("Descreva o dia (2 a 200 caracteres).")
	case !tamanho(in.FonteCitacao, 3, 300):
		return in, data, webx.Invalidf("Sem fonte não grava: cite a norma ou portaria (mínimo 3 caracteres).")
	case utf8.RuneCountInString(in.FonteURL) > 500 || utf8.RuneCountInString(in.Observacao) > 500:
		return in, data, webx.Invalidf("Link ou observação longo demais (máximo 500 caracteres).")
	case in.FonteURL != "" && !strings.HasPrefix(in.FonteURL, "https://") && !strings.HasPrefix(in.FonteURL, "http://"):
		return in, data, webx.Invalidf("O link da fonte deve começar com http:// ou https://.")
	}
	return in, data, nil
}
