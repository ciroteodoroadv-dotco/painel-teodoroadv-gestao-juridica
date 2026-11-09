package djen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL        = "https://comunica.pje.jus.br/api/v1/comunicacao"
	DefaultTimeout = 15 * time.Second
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
}

type QueryParams struct {
	NumeroOAB      string
	UFOAB          string
	SiglaTribunal  string
	DataInicio     string // YYYY-MM-DD
	DataFim        string // YYYY-MM-DD
	Pagina         int
	ItensPorPagina int
}

type ComunicacaoResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Items   []ComunicacaoItem `json:"items"`
	Count   int               `json:"count"`
}

type ComunicacaoItem struct {
	ID                   int    `json:"id"`
	NumeroProcesso       string `json:"numeroProcesso"`
	SiglaTribunal        string `json:"siglaTribunal"`
	OrgaoJulgador        string `json:"nomeOrgao"`
	TextoComunicacao     string `json:"texto"`
	TipoComunicacao      string `json:"tipoComunicacao"`
	DataDisponibilizacao string `json:"data_disponibilizacao"`
	NomeClasse           string `json:"nomeClasse"`
	Meio                 string `json:"meio"`
	NumeroPratica        int    `json:"numero_pratica"`
	Hash                 string `json:"hash"`
}

func (c *Client) BuscarIntimacoes(ctx context.Context, params QueryParams) (*ComunicacaoResponse, error) {
	u, err := url.Parse(BaseURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar URL base do DJEN: %w", err)
	}

	q := u.Query()
	if params.NumeroOAB != "" {
		q.Set("numeroOab", params.NumeroOAB)
	}
	if params.UFOAB != "" {
		q.Set("ufOab", params.UFOAB)
	}
	if params.SiglaTribunal != "" {
		q.Set("siglaTribunal", params.SiglaTribunal)
	}
	if params.DataInicio != "" {
		q.Set("dataDisponibilizacaoInicio", params.DataInicio)
	}
	if params.DataFim != "" {
		q.Set("dataDisponibilizacaoFim", params.DataFim)
	}
	if params.Pagina > 0 {
		q.Set("pagina", fmt.Sprintf("%d", params.Pagina))
	} else {
		q.Set("pagina", "1")
	}
	if params.ItensPorPagina > 0 {
		q.Set("itensPorPagina", fmt.Sprintf("%d", params.ItensPorPagina))
	} else {
		q.Set("itensPorPagina", "50")
	}

	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição DJEN: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "PainelTeodoroAdvocacia/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão com DJEN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resposta inválida do DJEN: status %d", resp.StatusCode)
	}

	var djenResp ComunicacaoResponse
	if err := json.NewDecoder(resp.Body).Decode(&djenResp); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON do DJEN: %w", err)
	}

	return &djenResp, nil
}
