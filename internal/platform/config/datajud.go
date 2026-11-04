package config

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Segredo é um texto que nunca aparece em log, em %v nem em %#v.
type Segredo string

func (Segredo) String() string   { return "[redigido]" }
func (Segredo) GoString() string { return "[redigido]" }

// DataJud é a configuração do monitor do DataJud (Plano 6). Vem DESLIGADO: só PAINEL_DATAJUD_ATIVO=true liga, e só depois
// do teste manual orientado. A chave é segredo (PAINEL_DATAJUD_KEY ou PAINEL_DATAJUD_KEY_FILE); ausente, a coleta fica
// "bloqueada" com a mensagem "Chave do DataJud não configurada.".
type DataJud struct {
	Ativo                   bool
	URL                     string
	Chave                   Segredo
	Intervalo               time.Duration // entre rodadas (padrão 24 h)
	IntervaloEntreProcessos time.Duration // pausa entre consultas (padrão 1 s)
	AlertaHoras             int           // padrão 36: o DataJud não é em tempo real
}

func loadDataJud(e Env, errs *[]error) DataJud {
	d := DataJud{URL: orDefault(e.Getenv("PAINEL_DATAJUD_URL"), "https://api-publica.datajud.cnj.jus.br"),
		Intervalo: 24 * time.Hour, IntervaloEntreProcessos: time.Second, AlertaHoras: 36}
	switch v := strings.ToLower(strings.TrimSpace(e.Getenv("PAINEL_DATAJUD_ATIVO"))); v {
	case "", "false", "0":
	case "true", "1":
		d.Ativo = true
	default:
		*errs = append(*errs, errors.New("PAINEL_DATAJUD_ATIVO inválida: use true ou false"))
	}
	if u, err := url.Parse(d.URL); err != nil || u.Scheme != "https" || u.Hostname() == "" {
		*errs = append(*errs, errors.New("PAINEL_DATAJUD_URL inválida: use um endereço https"))
	}
	if v := strings.TrimSpace(e.Getenv("PAINEL_DATAJUD_INTERVALO")); v != "" {
		x, err := time.ParseDuration(v)
		if err != nil || x < 15*time.Minute {
			*errs = append(*errs, errors.New("PAINEL_DATAJUD_INTERVALO inválida: use uma duração de pelo menos 15m (ex.: 24h)"))
		} else {
			d.Intervalo = x
		}
	}
	if v := strings.TrimSpace(e.Getenv("PAINEL_DATAJUD_PAUSA")); v != "" {
		x, err := time.ParseDuration(v)
		if err != nil || x < 500*time.Millisecond {
			*errs = append(*errs, errors.New("PAINEL_DATAJUD_PAUSA inválida: use pelo menos 500ms (ex.: 1s)"))
		} else {
			d.IntervaloEntreProcessos = x
		}
	}
	if v := strings.TrimSpace(e.Getenv("PAINEL_DATAJUD_ALERTA_HORAS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 336 {
			*errs = append(*errs, errors.New("PAINEL_DATAJUD_ALERTA_HORAS inválida: use um número de 1 a 336"))
		} else {
			d.AlertaHoras = n
		}
	}
	if k, err := secret(e, "PAINEL_DATAJUD_KEY"); err == nil { // ausente não é erro: a coleta fica bloqueada
		d.Chave = Segredo(k)
	} else if e.Getenv("PAINEL_DATAJUD_KEY_FILE") != "" {
		*errs = append(*errs, err) // arquivo indicado mas ilegível
	}
	return d
}
