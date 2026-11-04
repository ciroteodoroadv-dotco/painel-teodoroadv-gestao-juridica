package bcb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// BaseOficial é o endereço público da API de dados abertos do BCB.
const BaseOficial = "https://api.bcb.gov.br"

// Config do cliente.
type Config struct {
	BaseURL      string
	PermitirHTTP bool // só para teste com httptest.Server
	Timeout      time.Duration
	Pausa        time.Duration // espera mínima entre requisições (1 por série)
	MaxBytes     int64
	Transport    http.RoundTripper // só para teste
	Dormir       func(time.Duration)
}

// Cliente faz uma requisição por vez.
type Cliente struct {
	cfg  Config
	http *http.Client
	n    int
}

// Novo valida a configuração. Redirecionamento a outro host é recusado.
func Novo(cfg Config) (*Cliente, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseOficial
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !(cfg.PermitirHTTP && u.Scheme == "http")) {
		return nil, errors.New("bcb: a URL base precisa ser https")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 64 << 10
	}
	if cfg.Dormir == nil {
		cfg.Dormir = time.Sleep
	}
	host := u.Host
	hc := &http.Client{Timeout: cfg.Timeout, Transport: cfg.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != host || len(via) > 2 {
			return errors.New("redirecionamento recusado")
		}
		return nil
	}}
	return &Cliente{cfg: cfg, http: hc}, nil
}

var decimalRE = regexp.MustCompile(`^-?[0-9]{1,15}(\.[0-9]{1,10})?$`)

// Decimal normaliza "1.518,00", "5,4321" ou "0.52" para "1518.00", "5.4321", "0.52" (sem float).
func Decimal(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.Replace(s, ",", ".", 1)
	}
	if !decimalRE.MatchString(s) {
		return "", errors.New("bcb: valor inválido")
	}
	return s, nil
}

// Ultimo lê o último ponto da série. Em qualquer erro devolve Ponto vazio. Código fora de Series nem sai para a rede.
func (c *Cliente) Ultimo(ctx context.Context, codigo int) (Ponto, error) {
	if !conhecida(codigo) {
		return Ponto{}, errors.New("bcb: série desconhecida")
	}
	if c.n > 0 && c.cfg.Pausa > 0 {
		c.cfg.Dormir(c.cfg.Pausa)
	}
	c.n++
	endpoint := fmt.Sprintf("%s/dados/serie/bcdata.sgs.%d/dados/ultimos/1?formato=json", strings.TrimRight(c.cfg.BaseURL, "/"), codigo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Ponto{}, errors.New("bcb: requisição inválida")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "painel-teodoro")
	resp, err := c.http.Do(req)
	if err != nil {
		return Ponto{}, fmt.Errorf("bcb: falha de rede (%s)", classe(err))
	}
	defer func() { _ = resp.Body.Close() }()
	b, rerr := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxBytes+1))
	switch {
	case resp.StatusCode != http.StatusOK:
		return Ponto{}, fmt.Errorf("bcb: HTTP %d", resp.StatusCode)
	case rerr != nil:
		return Ponto{}, errors.New("bcb: falha ao ler a resposta")
	case int64(len(b)) > c.cfg.MaxBytes:
		return Ponto{}, errors.New("bcb: resposta maior que o limite")
	}
	var linhas []struct{ Data, Valor string }
	if err := json.Unmarshal(b, &linhas); err != nil {
		return Ponto{}, errors.New("bcb: resposta ilegível")
	}
	if len(linhas) == 0 {
		return Ponto{}, errors.New("bcb: série sem dados")
	}
	d, err := time.Parse("02/01/2006", linhas[len(linhas)-1].Data)
	if err != nil {
		return Ponto{}, errors.New("bcb: data inválida")
	}
	v, err := Decimal(linhas[len(linhas)-1].Valor)
	if err != nil {
		return Ponto{}, err
	}
	return Ponto{Data: d, Valor: v}, nil
}

func classe(err error) string {
	var ne interface{ Timeout() bool }
	switch {
	case errors.As(err, &ne) && ne.Timeout():
		return "tempo esgotado"
	case errors.Is(err, context.Canceled):
		return "cancelada"
	case strings.Contains(err.Error(), "redirecionamento recusado"):
		return "redirecionamento recusado"
	}
	return "conexão"
}
