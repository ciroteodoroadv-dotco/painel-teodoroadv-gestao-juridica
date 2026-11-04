package config

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Indices é a coleta dos índices do Banco Central (Plano 8). DESLIGADA por padrão: só PAINEL_INDICES_ATIVO=true liga,
// e só depois do teste manual orientado.
type Indices struct {
	Ativo        bool
	URL          string
	Intervalo    time.Duration // entre rodadas (padrão 24 h)
	ValidadeDias int           // além disso o valor aparece "desatualizado" (padrão 40: IPCA, INPC e IGP-M são mensais)
}

// CEP é a consulta de endereço pelo CEP (ViaCEP). DESLIGADA por padrão (PAINEL_CEP_ATIVO).
type CEP struct {
	Ativo bool
	URL   string
}

func ligado(e Env, nome string, errs *[]error) bool {
	switch v := strings.ToLower(strings.TrimSpace(e.Getenv(nome))); v {
	case "", "false", "0":
		return false
	case "true", "1":
		return true
	default:
		*errs = append(*errs, errors.New(nome+" inválida: use true ou false"))
		return false
	}
}

func urlHTTPS(e Env, nome, padrao string, errs *[]error) string {
	u := orDefault(e.Getenv(nome), padrao)
	if p, err := url.Parse(u); err != nil || p.Scheme != "https" || p.Hostname() == "" {
		*errs = append(*errs, errors.New(nome+" inválida: use um endereço https"))
	}
	return u
}

func loadIndices(e Env, errs *[]error) Indices {
	i := Indices{Ativo: ligado(e, "PAINEL_INDICES_ATIVO", errs), URL: urlHTTPS(e, "PAINEL_INDICES_URL", "https://api.bcb.gov.br", errs),
		Intervalo: 24 * time.Hour, ValidadeDias: 40}
	if v := strings.TrimSpace(e.Getenv("PAINEL_INDICES_INTERVALO")); v != "" {
		x, err := time.ParseDuration(v)
		if err != nil || x < time.Hour {
			*errs = append(*errs, errors.New("PAINEL_INDICES_INTERVALO inválida: use uma duração de pelo menos 1h (ex.: 24h)"))
		} else {
			i.Intervalo = x
		}
	}
	if v := strings.TrimSpace(e.Getenv("PAINEL_INDICES_VALIDADE_DIAS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 400 {
			*errs = append(*errs, errors.New("PAINEL_INDICES_VALIDADE_DIAS inválida: use um número de 1 a 400"))
		} else {
			i.ValidadeDias = n
		}
	}
	return i
}

func loadCEP(e Env, errs *[]error) CEP {
	return CEP{Ativo: ligado(e, "PAINEL_CEP_ATIVO", errs), URL: urlHTTPS(e, "PAINEL_CEP_URL", "https://viacep.com.br", errs)}
}
