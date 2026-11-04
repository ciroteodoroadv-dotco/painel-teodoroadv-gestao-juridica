// Package datajud é o cliente isolado da API pública do DataJud (CNJ). Só lê. Não grava em banco nem conhece o painel:
// recebe o alias do tribunal e o número CNJ e devolve as movimentações. Nenhum teste fala com o servidor real.
package datajud

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"painelteodoro/internal/monitor/cnj"
)

// BaseOficial é o endereço público do DataJud.
const BaseOficial = "https://api-publica.datajud.cnj.jus.br"

// msgErro é um erro cujo texto é a frase mostrada ao usuário (termina com ponto).
type msgErro string

func (e msgErro) Error() string { return string(e) }

// MsgSemChave é o texto do estado quando a chave não foi configurada.
const MsgSemChave = "Chave do DataJud não configurada."

// Config do cliente. A chave só vem por função (lida de segredo em arquivo/variável) e nunca é impressa.
type Config struct {
	BaseURL       string
	PermitirHTTP  bool // só para teste com httptest.Server
	Chave         func() string
	Timeout       time.Duration
	Intervalo     time.Duration // espera mínima entre consultas
	EsperaTaxa    time.Duration // espera no 429
	MaxBytes      int64
	MaxTentativas int
	Transport     http.RoundTripper // só para teste
	Dormir        func(time.Duration)
}

// String não mostra a chave.
func (c Config) String() string {
	return fmt.Sprintf("datajud.Config{BaseURL:%q, chave:%s}", c.BaseURL, map[bool]string{true: "função definida", false: "ausente"}[c.Chave != nil])
}

// GoString idem (para %#v).
func (c Config) GoString() string { return c.String() }

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
		return nil, errors.New("datajud: a URL base precisa ser https")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.EsperaTaxa <= 0 {
		cfg.EsperaTaxa = 60 * time.Second
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = 8 << 20
	}
	if cfg.MaxTentativas <= 0 {
		cfg.MaxTentativas = 3
	}
	if cfg.Dormir == nil {
		cfg.Dormir = time.Sleep
	}
	host := u.Host
	hc := &http.Client{Timeout: cfg.Timeout, Transport: cfg.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != host || len(via) > 3 {
			return errors.New("redirecionamento recusado")
		}
		return nil
	}}
	return &Cliente{cfg: cfg, http: hc}, nil
}

// Consultar busca o processo pelo número CNJ no endpoint do tribunal (alias). Em qualquer erro devolve Resultado vazio.
func (c *Cliente) Consultar(ctx context.Context, alias, numeroCNJ string) (Resultado, error) {
	chave := ""
	if c.cfg.Chave != nil {
		chave = strings.TrimSpace(c.cfg.Chave())
	}
	if chave == "" {
		return Resultado{}, msgErro(MsgSemChave)
	}
	d := cnj.Digitos(numeroCNJ)
	if len(d) != 20 || alias == "" || strings.ContainsAny(alias, "/?# ") {
		return Resultado{}, errors.New("datajud: alias ou número inválido")
	}
	corpo, _ := json.Marshal(map[string]any{"size": 20, "query": map[string]any{"match": map[string]any{"numeroProcesso": d}}})
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/api_publica_" + alias + "/_search"
	if c.n > 0 && c.cfg.Intervalo > 0 {
		c.cfg.Dormir(c.cfg.Intervalo)
	}
	c.n++
	for tentativa := 1; ; tentativa++ {
		if err := ctx.Err(); err != nil {
			return Resultado{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(corpo))
		if err != nil {
			return Resultado{}, errors.New("datajud: requisição inválida")
		}
		req.Header.Set("Authorization", "APIKey "+chave)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "painel-teodoro")
		resp, err := c.http.Do(req)
		if err != nil {
			return Resultado{}, fmt.Errorf("datajud: falha de rede (%s)", classe(err))
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			if tentativa >= c.cfg.MaxTentativas {
				return Resultado{}, errors.New("datajud: limite de requisições (429) persistente")
			}
			c.cfg.Dormir(c.cfg.EsperaTaxa)
			continue
		}
		b, rerr := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxBytes+1))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode != http.StatusOK:
			return Resultado{}, fmt.Errorf("datajud: HTTP %d", resp.StatusCode)
		case rerr != nil:
			return Resultado{}, errors.New("datajud: falha ao ler a resposta")
		case int64(len(b)) > c.cfg.MaxBytes:
			return Resultado{}, errors.New("datajud: resposta maior que o limite")
		}
		return interpretar(b)
	}
}

// classe resume o erro de rede sem expor a URL.
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

// TemChave diz se há chave configurada (sem fazer nenhuma requisição).
func (c *Cliente) TemChave() bool {
	return c.cfg.Chave != nil && strings.TrimSpace(c.cfg.Chave()) != ""
}
