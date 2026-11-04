package prazos

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos/calculo"
	"painelteodoro/internal/prazos/calendario"
)

// Calendarios é o que a calculadora precisa do cadastro do calendário forense.
type Calendarios interface {
	Calendarios(ctx context.Context) ([]calendario.Calendario, error)
	Carregar(ctx context.Context, id string) (calculo.Calendario, error)
}

// regrasDoB traduz o valor do rádio do painel B para a regra do cálculo.
var regrasDoB = map[string]calculo.Regra{"uteis": calculo.CPCUteis, "corridos": calculo.CPPCorridos,
	"material-civel": calculo.MaterialCivel, "material-penal": calculo.MaterialPenal}

var rotuloRegra = map[calculo.Regra]string{calculo.CPCUteis: "CPC, dias úteis", calculo.CPPCorridos: "CPP, dias corridos",
	calculo.MaterialCivel: "prazo material cível", calculo.MaterialPenal: "prazo material penal"}

func (h *Handler) agora() time.Time {
	if h.Agora != nil {
		return h.Agora()
	}
	return time.Now()
}

func (h *Handler) calculadoraDrawer(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	cals, err := h.Cal.Calendarios(r.Context())
	if err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
		return
	}
	var locais []calendario.Calendario
	for _, c := range cals {
		if c.Ambito != "nacional" { // o nacional é SEMPRE aplicado; não é escolha
			locais = append(locais, c)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = fragmento("calculadora.html").ExecuteTemplate(w, "calculadora",
		map[string]any{"Hoje": webx.Hoje(h.agora()).Format("2006-01-02"), "Calendarios": locais})
}

func (h *Handler) erroCalculo(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = fragmento("resultado.html").ExecuteTemplate(w, "erro", msg)
}

type resultadoView struct {
	Venc, Frase                                      string
	Premissas, Alternativas, Duvidas, NaoConsiderado []string
	Excluidos                                        []calculo.DiaNaoUtil
	Validada                                         bool
	Link                                             string
}

func frase(n int) string {
	switch {
	case n == 0:
		return "vence hoje"
	case n > 0:
		return fmt.Sprintf("faltam %d dia(s) útil(eis)", n)
	}
	return fmt.Sprintf("venceu há %d dia(s) útil(eis)", -n)
}

func (h *Handler) calcular(w http.ResponseWriter, r *http.Request, s *auth.Session) {
	v := r.PostFormValue
	evento, _ := webx.DataISO(v("base")) // data inválida = zero; o cálculo responde "Data do evento inválida."
	dias, err := strconv.Atoi(strings.TrimSpace(v("dias")))
	if err != nil {
		dias = 0
	}
	regra, ok := regrasDoB[v("tipo")]
	if !ok {
		regra = calculo.Regra(v("tipo")) // desconhecida: o cálculo recusa com a frase certa
	}
	calID := strings.TrimSpace(v("calendario"))
	if calID != "" && !webx.UUID(calID) {
		h.erroCalculo(w, http.StatusBadRequest, "Calendário inválido.")
		return
	}
	cal, err := h.Cal.Carregar(r.Context(), calID)
	if err != nil {
		h.erroCalculo(w, http.StatusInternalServerError, "Falha ao calcular o prazo.")
		return
	}
	hoje := webx.Hoje(h.agora())
	e := calculo.Entrada{Evento: evento, Modo: calculo.Modo(v("modo")), Regra: regra, Dias: dias, Cal: cal,
		Hoje: time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC)}
	res, err := calculo.Calcular(e)
	if err != nil {
		h.erroCalculo(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := audit.RecordPool(r.Context(), h.S.Pool, audit.Event{Actor: s.UserID, Action: "prazo.calculado", TargetType: "prazo",
		TargetID: "calculadora", Result: audit.OK, RequestID: w.Header().Get("X-Request-Id"),
		Meta: map[string]string{"regra": string(regra), "modo": v("modo"), "dias": strconv.Itoa(dias), "calendario": map[bool]string{true: "local", false: "nacional"}[calID != ""]}}); err != nil {
		h.erroCalculo(w, http.StatusInternalServerError, "Falha ao calcular o prazo.")
		return
	}
	venc := *res.Vencimento
	view := resultadoView{Venc: venc.Format("02/01/2006"), Frase: frase(res.DiasUteisAteVencimento), Premissas: res.Premissas,
		Alternativas: res.Alternativas, Duvidas: res.Duvidas, NaoConsiderado: res.NaoConsiderado, Excluidos: res.Excluidos,
		Validada: h.Validada, Link: linkCriarPrazo(e, venc)}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("resultado.html").ExecuteTemplate(w, "resultado", view); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}

// linkCriarPrazo leva ao formulário de Prazos JÁ preenchido. Não cria nada: o prazo nasce A CONFERIR quando salvo.
func linkCriarPrazo(e calculo.Entrada, venc time.Time) string {
	fund := fmt.Sprintf("Calculadora (A CONFERIR): %s, %d dia(s), evento em %s (%s); calendário: %s. Conferir no processo e no tribunal.",
		rotuloRegra[e.Regra], e.Dias, e.Evento.Format("02/01/2006"), map[calculo.Modo]string{calculo.DJe: "Diário", calculo.Ciencia: "ciência"}[e.Modo],
		map[bool]string{true: "nacional + tribunal cadastrado", false: "só feriados nacionais"}[e.Cal != nil])
	if utf8.RuneCountInString(fund) > 300 {
		fund = string([]rune(fund)[:300])
	}
	q := url.Values{"data_fatal": {venc.Format("2006-01-02")}, "fonte": {"calculadora"}, "fundamento": {fund}}
	return "/prazos/novo?" + q.Encode()
}
