package bcb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	BaseURL        = "https://api.bcb.gov.br/dados/serie/bcdata.sgs.%d/dados"
	DefaultTimeout = 15 * time.Second

	SerieSelic = 11  // Taxa Selic diária (%)
	SerieINPC  = 188 // INPC mensal
	SerieIPCA  = 433 // IPCA mensal
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
	}
}

type DadoSerieRaw struct {
	Data  string `json:"data"`  // DD/MM/YYYY
	Valor string `json:"valor"` // Valor percentual em string
}

type DadoSerie struct {
	Data  time.Time
	Valor float64
}

func (c *Client) BuscarSerie(ctx context.Context, codigoSerie int, dataInicio, dataFim string) ([]DadoSerie, error) {
	urlStr := fmt.Sprintf(BaseURL+"?formato=json", codigoSerie)
	if dataInicio != "" && dataFim != "" {
		urlStr += fmt.Sprintf("&dataInicial=%s&dataFinal=%s", dataInicio, dataFim)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar requisição para BCB [série %d]: %w", codigoSerie, err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "PainelTeodoroAdvocacia/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro de conexão com BCB: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resposta inválida do BCB [série %d]: status %d", codigoSerie, resp.StatusCode)
	}

	var raws []DadoSerieRaw
	if err := json.NewDecoder(resp.Body).Decode(&raws); err != nil {
		return nil, fmt.Errorf("falha ao decodificar JSON do BCB: %w", err)
	}

	var resultados []DadoSerie
	for _, r := range raws {
		t, err := time.Parse("02/01/2006", r.Data)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(r.Valor, 64)
		if err != nil {
			continue
		}
		resultados = append(resultados, DadoSerie{
			Data:  t,
			Valor: v,
		})
	}

	return resultados, nil
}
