// Package webx reúne utilidades comuns às telas de domínio: erros padronizados, htmx e paginação.
package webx

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Erros que os domínios devolvem e as telas traduzem em respostas.
var (
	ErrNotFound = errors.New("registro não encontrado")
	ErrConflict = errors.New("o registro foi alterado por outra pessoa; recarregue e tente de novo")
)

// Invalid é erro de validação com mensagem própria para o usuário.
type Invalid struct{ Msg string }

func (e Invalid) Error() string { return e.Msg }

// Invalidf cria um erro de validação.
func Invalidf(msg string) error { return Invalid{Msg: msg} }

// Opcao é um valor/rótulo de lista suspensa, compartilhado entre domínios.
type Opcao struct{ Valor, Rotulo string }

// IsHX diz se a requisição veio do htmx (devolve só o fragmento).
func IsHX(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

// Like escapa % _ \ para uso em ILIKE e envolve com %.
func Like(q string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(strings.TrimSpace(q)) + "%"
}

// PageSize é o tamanho fixo de página das listas.
const PageSize = 25

// Page lê ?pag= (1,2,3…) e devolve página, limite e deslocamento.
func Page(r *http.Request) (page, limit, offset int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("pag"))
	if page < 1 {
		page = 1
	}
	if page > 10000 {
		page = 10000
	}
	return page, PageSize + 1, (page - 1) * PageSize // +1 para saber se há próxima
}

// Int lê um inteiro de formulário (0 se inválido).
func Int(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }

// Nullable devolve nil para string vazia (grava NULL no banco).
func Nullable(s string) *string {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	return &s
}

// SP é o fuso do escritório. Datas e horas da agenda são digitadas e exibidas nele.
func SP() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.FixedZone("BRT", -3*3600)
	}
	return loc
}

// ParseLocal lê "2006-01-02T15:04" (campo datetime-local) no fuso do escritório e devolve em UTC.
func ParseLocal(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(s), SP())
	return t.UTC(), err
}

// FormatLocal devolve "2006-01-02T15:04" no fuso do escritório (para preencher datetime-local).
func FormatLocal(t time.Time) string { return t.In(SP()).Format("2006-01-02T15:04") }

// Hoje devolve a data de hoje no fuso do escritório, à meia-noite UTC (para comparar com colunas DATE).
func Hoje(now time.Time) time.Time {
	n := now.In(SP())
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}
