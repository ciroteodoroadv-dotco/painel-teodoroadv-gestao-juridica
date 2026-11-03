package djen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Config do cliente. BaseURL é fixa (vem da configuração do painel, nunca do usuário) e precisa ser https.
type Config struct {
	BaseURL                     string
	PermitirHTTP                bool // só para teste com httptest.Server; produção exige https
	Timeout                     time.Duration
	IntervaloMinimo, EsperaTaxa time.Duration
	MaxBytesResposta            int64
	MaxPaginas, MaxResultados   int
	MaxTentativasTaxa           int
	ItensPorPagina              int
	Transport                   http.RoundTripper // só para teste; nil usa o transporte padrão
	Agora                       func() time.Time
	Dormir                      func(time.Duration)
}

// Comunicacao é uma comunicação do DJEN já normalizada.
type Comunicacao struct {
	ID, Tribunal, Orgao, Processo, Tipo string
	DataPublicacao                      time.Time // data civil (meia-noite UTC do dia)
	Texto, Link                         string
}

// Resultado da coleta. Parcial só é verdadeiro quando um único dia ainda estourou o teto de resultados.
type Resultado struct {
	Itens   []Comunicacao
	Parcial bool
	Motivo  string
}

// Cliente faz as requisições, uma por vez.
type Cliente struct {
	cfg  Config
	http *http.Client
	n    int // requisições já feitas (a primeira não espera o intervalo)
}

var (
	soDigitos = regexp.MustCompile(`^[0-9]{1,10}$`)
	soLetras  = regexp.MustCompile(`^[A-Za-z]{2}$`)
)

// Novo valida a configuração e prepara o cliente. Redirecionamento a outro host é recusado.
func Novo(cfg Config) (*Cliente, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && !(cfg.PermitirHTTP && u.Scheme == "http")) {
		return nil, errors.New("djen: a URL base precisa ser https")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.EsperaTaxa <= 0 {
		cfg.EsperaTaxa = 60 * time.Second
	}
	if cfg.MaxBytesResposta <= 0 {
		cfg.MaxBytesResposta = 8 << 20
	}
	if cfg.MaxPaginas <= 0 {
		cfg.MaxPaginas = 100
	}
	if cfg.MaxResultados <= 0 {
		cfg.MaxResultados = 10000
	}
	if cfg.MaxTentativasTaxa <= 0 {
		cfg.MaxTentativasTaxa = 3
	}
	if cfg.ItensPorPagina <= 0 {
		cfg.ItensPorPagina = 100
	}
	if cfg.Dormir == nil {
		cfg.Dormir = time.Sleep
	}
	host := u.Host
	hc := &http.Client{Timeout: cfg.Timeout, Transport: cfg.Transport, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if req.URL.Host != host || len(via) > 3 {
			return errors.New("djen: redirecionamento recusado")
		}
		return nil
	}}
	return &Cliente{cfg: cfg, http: hc}, nil
}

// Coletar devolve todas as comunicações da OAB entre desde e ate (datas civis, inclusive). Em qualquer erro, devolve
// Resultado vazio: um pedaço da coleta nunca pode ser tomado pela coleta inteira.
func (c *Cliente) Coletar(ctx context.Context, oab, uf string, desde, ate time.Time) (Resultado, error) {
	oab, uf = strings.TrimSpace(oab), strings.ToUpper(strings.TrimSpace(uf))
	if !soDigitos.MatchString(oab) || !soLetras.MatchString(uf) {
		return Resultado{}, errors.New("djen: OAB ou UF inválida")
	}
	desde, ate = dataCivil(desde), dataCivil(ate)
	if ate.Before(desde) {
		return Resultado{}, errors.New("djen: janela de datas invertida")
	}
	itens, parcial, motivo, err := c.janela(ctx, oab, uf, desde, ate)
	if err != nil {
		return Resultado{}, err
	}
	return Resultado{Itens: itens, Parcial: parcial, Motivo: motivo}, nil
}

// janela coleta [desde, ate]; se estourar o teto de resultados e houver mais de um dia, refaz dia a dia.
func (c *Cliente) janela(ctx context.Context, oab, uf string, desde, ate time.Time) (itens []Comunicacao, parcial bool, motivo string, err error) {
	itens, estourou, err := c.paginas(ctx, oab, uf, desde, ate)
	if err != nil {
		return nil, false, "", err
	}
	if !estourou {
		return itens, false, "", nil
	}
	if desde.Equal(ate) {
		return itens, true, "o DJEN devolveu o teto de resultados num único dia (" + desde.Format("02/01/2006") + "); pode haver comunicações não lidas", nil
	}
	var todos []Comunicacao
	for d := desde; !d.After(ate); d = d.AddDate(0, 0, 1) {
		it, p, m, err := c.janela(ctx, oab, uf, d, d)
		if err != nil {
			return nil, false, "", err
		}
		todos = append(todos, it...)
		if p {
			parcial, motivo = true, m
		}
	}
	return todos, parcial, motivo, nil
}

type itemJSON map[string]any

type paginaJSON struct {
	Count *int            `json:"count"`
	Items json.RawMessage `json:"items"`
}

// paginas lê todas as páginas de uma janela. estourou=true quando o DJEN anuncia o teto (os itens lidos são descartáveis).
func (c *Cliente) paginas(ctx context.Context, oab, uf string, desde, ate time.Time) (out []Comunicacao, estourou bool, err error) {
	vistos := map[string]bool{}
	for pag := 1; pag <= c.cfg.MaxPaginas; pag++ {
		corpo, err := c.pedir(ctx, oab, uf, pag, desde, ate)
		if err != nil {
			return nil, false, err
		}
		if corpo.Count != nil && *corpo.Count >= c.cfg.MaxResultados {
			return nil, true, nil
		}
		if len(corpo.Items) == 0 || string(corpo.Items) == "null" {
			return nil, false, errors.New("djen: resposta sem a lista 'items'; coleta não confirmada")
		}
		var brutos []itemJSON
		if err := json.Unmarshal(corpo.Items, &brutos); err != nil {
			return nil, false, errors.New("djen: lista 'items' inválida")
		}
		novos := 0
		for _, b := range brutos {
			cm, err := normalizar(b)
			if err != nil {
				return nil, false, err
			}
			h := Hash(cm)
			if vistos[h] {
				continue
			}
			vistos[h] = true
			novos++
			out = append(out, cm)
		}
		if corpo.Count == nil { // contagem desconhecida: página incompleta encerra
			if len(brutos) < c.cfg.ItensPorPagina {
				return out, false, nil
			}
		} else if len(vistos) >= *corpo.Count {
			return out, false, nil
		}
		if novos == 0 {
			return nil, false, errors.New("djen: paginação repetida ou vazia antes do fim; coleta não confirmada")
		}
	}
	return nil, false, errors.New("djen: limite de páginas atingido; coleta não confirmada")
}

// pedir faz uma requisição (esperando o intervalo mínimo desde a anterior) e repete a mesma página no 429.
func (c *Cliente) pedir(ctx context.Context, oab, uf string, pagina int, desde, ate time.Time) (paginaJSON, error) {
	u, _ := url.Parse(c.cfg.BaseURL)
	q := u.Query()
	q.Set("numeroOab", oab)
	q.Set("ufOab", uf)
	q.Set("pagina", strconv.Itoa(pagina))
	q.Set("itensPorPagina", strconv.Itoa(c.cfg.ItensPorPagina))
	q.Set("dataDisponibilizacaoInicio", desde.Format("2006-01-02"))
	q.Set("dataDisponibilizacaoFim", ate.Format("2006-01-02"))
	u.RawQuery = q.Encode()
	for tentativa := 0; ; tentativa++ {
		if err := ctx.Err(); err != nil {
			return paginaJSON{}, err
		}
		if c.n > 0 && c.cfg.IntervaloMinimo > 0 {
			c.cfg.Dormir(c.cfg.IntervaloMinimo)
		}
		c.n++
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return paginaJSON{}, err
		}
		req.Header.Set("User-Agent", "painel-teodoro")
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return paginaJSON{}, fmt.Errorf("djen: falha de rede (%s)", classeDoErro(err))
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			_ = resp.Body.Close()
			if tentativa+1 >= c.cfg.MaxTentativasTaxa {
				return paginaJSON{}, errors.New("djen: limite de requisições (429) persistente")
			}
			c.cfg.Dormir(c.cfg.EsperaTaxa)
			continue
		}
		esgotou := resp.Header.Get("X-Ratelimit-Remaining") == "0"
		b, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxBytesResposta+1))
		_ = resp.Body.Close()
		switch {
		case resp.StatusCode != http.StatusOK:
			return paginaJSON{}, fmt.Errorf("djen: HTTP %d", resp.StatusCode)
		case err != nil:
			return paginaJSON{}, errors.New("djen: falha ao ler a resposta")
		case int64(len(b)) > c.cfg.MaxBytesResposta:
			return paginaJSON{}, errors.New("djen: resposta maior que o limite")
		}
		var p paginaJSON
		if err := json.Unmarshal(b, &p); err != nil {
			return paginaJSON{}, errors.New("djen: JSON inválido")
		}
		if esgotou { // o servidor avisou que acabou a cota: espera antes da próxima
			c.cfg.Dormir(c.cfg.EsperaTaxa)
		}
		return p, nil
	}
}

// classeDoErro resume o erro de rede sem expor a URL (que leva o número da OAB).
func classeDoErro(err error) string {
	var ne interface{ Timeout() bool }
	if errors.As(err, &ne) && ne.Timeout() {
		return "tempo esgotado"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelada"
	}
	return "conexão"
}
