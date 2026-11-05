// Package cep consulta endereço pelo CEP no ViaCEP (público, não oficial). O navegador nunca fala com a internet: o
// painel faz a consulta. Só os 8 dígitos do CEP saem; nenhum dado do usuário. Nenhum teste fala com o servidor real.
package cep

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BaseOficial é o endereço público do ViaCEP.
const BaseOficial = "https://viacep.com.br"

// Fonte é o texto exibido ao usuário, igual ao do painel B.
const Fonte = "ViaCEP (público, não oficial)"

// Erros que a tela traduz em mensagem.
var (
	ErrCEPInvalido   error = msgErro("Informe um CEP com oito dígitos.")
	ErrNaoEncontrado error = msgErro("CEP não encontrado.")
	ErrIndisponivel  error = msgErro("Não foi possível consultar o CEP agora. Tente de novo mais tarde ou digite o endereço.")
)

// msgErro é um erro cujo texto é a frase mostrada ao usuário (por isso começa com maiúscula e termina com ponto).
type msgErro string

func (e msgErro) Error() string { return string(e) }

// Endereco é o que o formulário preenche.
type Endereco struct{ Logradouro, Bairro, Municipio, UF string }

// Config do cliente.
type Config struct {
	BaseURL      string
	PermitirHTTP bool // só para teste
	Timeout      time.Duration
	MaxBytes     int64
	Transport    http.RoundTripper // só para teste
}

// Cliente consulta o ViaCEP.
type Cliente struct {
	base string
	max  int64
	http *http.Client
}

// NovoCliente valida a configuração. Redirecionamento a outro host é recusado.
func NovoCliente(cfg Config) (*Cliente, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseOficial
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && (!cfg.PermitirHTTP || u.Scheme != "http")) {
		return nil, errors.New("cep: a URL base precisa ser https")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 8 * time.Second
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 16 << 10
	}
	host := u.Host
	return &Cliente{base: strings.TrimRight(cfg.BaseURL, "/"), max: cfg.MaxBytes, http: &http.Client{Timeout: cfg.Timeout, Transport: cfg.Transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Host != host || len(via) > 2 {
				return errors.New("redirecionamento recusado")
			}
			return nil
		}}}, nil
}

// Digitos devolve só os dígitos de s.
func Digitos(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func oitoDigitos(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

type resposta struct {
	Logradouro string          `json:"logradouro"`
	Bairro     string          `json:"bairro"`
	Localidade string          `json:"localidade"`
	UF         string          `json:"uf"`
	Erro       json.RawMessage `json:"erro"`
}

// Buscar consulta o CEP (exatamente 8 dígitos). Em qualquer erro devolve Endereco vazio: nunca dado inventado.
func (c *Cliente) Buscar(ctx context.Context, cep string) (Endereco, error) {
	if !oitoDigitos(cep) {
		return Endereco{}, ErrCEPInvalido
	}
	// #nosec G704 -- falso positivo: a base é a URL https da configuração (PAINEL_CEP_URL, validada em NovoCliente) e o CEP já foi validado como 8 dígitos.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/ws/"+cep+"/json/", nil)
	if err != nil {
		return Endereco{}, ErrIndisponivel
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "painel-teodoro")
	// #nosec G704 -- idem: destino fixo pela configuração; só os 8 dígitos do CEP variam.
	resp, err := c.http.Do(req)
	if err != nil {
		return Endereco{}, ErrIndisponivel
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Endereco{}, ErrIndisponivel
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, c.max+1))
	if err != nil || int64(len(b)) > c.max {
		return Endereco{}, ErrIndisponivel
	}
	var r resposta
	if json.Unmarshal(b, &r) != nil {
		return Endereco{}, ErrIndisponivel
	}
	if e := strings.Trim(string(r.Erro), `" `); e == "true" {
		return Endereco{}, ErrNaoEncontrado
	}
	e := Endereco{Logradouro: limpa(r.Logradouro), Bairro: limpa(r.Bairro), Municipio: limpa(r.Localidade), UF: strings.ToUpper(limpa(r.UF))}
	if e == (Endereco{}) {
		return Endereco{}, ErrIndisponivel
	}
	return e, nil
}

func limpa(s string) string {
	s = strings.TrimSpace(strings.ToValidUTF8(s, ""))
	if r := []rune(s); len(r) > 200 {
		s = string(r[:200])
	}
	return s
}
