package datajud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	BaseURL = "https://api-publica.datajud.cnj.jus.br"
	// #nosec G101 -- Chave publica padrao fornecida pelo CNJ para a API do DataJud
	DefaultAPIKey  = "APIKey cjFwYXJ0aWNpcGFudGU6c2VjcmV0bw=="
	DefaultTimeout = 15 * time.Second
)

type Client struct {
	httpClient *http.Client
	apiKey     string
}

func NewClient(apiKey string) *Client {
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		apiKey:     apiKey,
	}
}

type SearchQuery struct {
	Query QueryBody `json:"query"`
}

type QueryBody struct {
	Match MatchBody `json:"match"`
}

type MatchBody struct {
	NumeroProcesso string `json:"numeroProcesso"`
}

type Response struct {
	Hits HitsBody `json:"hits"`
}

type HitsBody struct {
	Total TotalHits `json:"total"`
	Hits  []HitItem `json:"hits"`
}

type TotalHits struct {
	Value int `json:"value"`
}

type HitItem struct {
	Source ProcessoSource `json:"_source"`
}

type ProcessoSource struct {
	NumeroProcesso            string        `json:"numeroProcesso"`
	Classe                    DetalheItem   `json:"classe"`
	Sistema                   DetalheItem   `json:"sistema"`
	Formatacao                string        `json:"formatacao"`
	DataHoraUltimaAtualizacao string        `json:"dataHoraUltimaAtualizacao"`
	OrgaoJulgador             OrgaoJulgador `json:"orgaoJulgador"`
	Movimentos                []Movimento   `json:"movimentos"`
	Assuntos                  []DetalheItem `json:"assuntos"`
}

type DetalheItem struct {
	Codigo int    `json:"codigo"`
	Nome   string `json:"nome"`
}

type OrgaoJulgador struct {
	Codigo int    `json:"codigo"`
	Nome   string `json:"nome"`
}

type Movimento struct {
	Codigo                int              `json:"codigo"`
	Nome                  string           `json:"nome"`
	DataHora              string           `json:"dataHora"`
	ComplementosTabelados []map[string]any `json:"complementosTabelados,omitempty"`
}

func (c *Client) BuscarProcesso(ctx context.Context, tribunal string, numeroCNJ string) (*ProcessoSource, error) {
	tribunalClean := strings.ToLower(strings.TrimSpace(tribunal))
	endpoint := fmt.Sprintf("%s/api_publica_%s/_search", BaseURL, tribunalClean)

	numLimpo := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, numeroCNJ)

	reqPayload := SearchQuery{
		Query: QueryBody{
			Match: MatchBody{
				NumeroProcesso: numLimpo,
			},
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("falha ao serializar payload DataJud: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição DataJud: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão com DataJud [%s]: %w", tribunalClean, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resposta inválida do DataJud [%s]: status %d", tribunalClean, resp.StatusCode)
	}

	var dataJudResp Response
	if err := json.NewDecoder(resp.Body).Decode(&dataJudResp); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON do DataJud: %w", err)
	}

	if len(dataJudResp.Hits.Hits) == 0 {
		return nil, fmt.Errorf("processo %s não encontrado no DataJud [%s]", numeroCNJ, tribunalClean)
	}

	return &dataJudResp.Hits.Hits[0].Source, nil
}
