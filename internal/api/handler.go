package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"painelteodoro/internal/auth"
)

type Handler struct {
	A    *auth.Auth
	Pool *pgxpool.Pool
}

func NewHandler(a *auth.Auth, pool *pgxpool.Pool) *Handler {
	return &Handler{A: a, Pool: pool}
}

func (h *Handler) Register(rt *auth.Router) {
	rt.Handle("GET /api/v1/processos", auth.PermProcRead, h.ListProcessos)
	rt.Handle("GET /api/v1/processos/{id}", auth.PermProcRead, h.GetProcesso)
	rt.Handle("GET /api/v1/intimacoes", auth.PermProcRead, h.ListIntimacoes)
	rt.Handle("GET /api/v1/indices", auth.PermProcRead, h.GetIndices)
}

type ProcessoSummary struct {
	ID            int64  `json:"id"`
	NumeroCNJ     string `json:"numero_cnj"`
	Tribunal      string `json:"tribunal"`
	OrgaoJulgador string `json:"orgao_julgador"`
	Status        string `json:"status"`
}

func (h *Handler) ListProcessos(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	ctx := r.Context()
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	sqlQuery := `
SELECT id, numero_cnj, COALESCE(tribunal, ''), COALESCE(orgao_julgador, ''), status
FROM processos
WHERE ($1 = '' OR numero_cnj ILIKE $2 OR tribunal ILIKE $2 OR orgao_julgador ILIKE $2)
ORDER BY id DESC
LIMIT 50
`

	searchParam := "%" + query + "%"
	rows, err := h.Pool.Query(ctx, sqlQuery, query, searchParam)
	if err != nil {
		Error(w, http.StatusInternalServerError, "erro ao consultar processos")
		return
	}
	defer rows.Close()

	var lista []ProcessoSummary
	for rows.Next() {
		var p ProcessoSummary
		if err := rows.Scan(&p.ID, &p.NumeroCNJ, &p.Tribunal, &p.OrgaoJulgador, &p.Status); err == nil {
			lista = append(lista, p)
		}
	}

	if lista == nil {
		lista = []ProcessoSummary{}
	}

	JSON(w, http.StatusOK, map[string]any{
		"data":  lista,
		"count": len(lista),
	})
}

func (h *Handler) GetProcesso(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	ctx := r.Context()
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		Error(w, http.StatusBadRequest, "ID de processo invalido")
		return
	}

	var p ProcessoSummary
	sqlQuery := `
SELECT id, numero_cnj, COALESCE(tribunal, ''), COALESCE(orgao_julgador, ''), status
FROM processos
WHERE id = $1
`
	err = h.Pool.QueryRow(ctx, sqlQuery, id).Scan(&p.ID, &p.NumeroCNJ, &p.Tribunal, &p.OrgaoJulgador, &p.Status)
	if err != nil {
		Error(w, http.StatusNotFound, "processo nao encontrado")
		return
	}

	JSON(w, http.StatusOK, p)
}

func (h *Handler) ListIntimacoes(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	ctx := r.Context()

	sqlQuery := `
SELECT id, COALESCE(sigla_tribunal, ''), COALESCE(nome_orgao, ''), COALESCE(texto, ''), criado_em
FROM intimacoes
ORDER BY id DESC
LIMIT 50
`
	rows, err := h.Pool.Query(ctx, sqlQuery)
	if err != nil {
		Error(w, http.StatusInternalServerError, "erro ao consultar intimacoes")
		return
	}
	defer rows.Close()

	type IntimacaoItem struct {
		ID       int64     `json:"id"`
		Tribunal string    `json:"tribunal"`
		Orgao    string    `json:"orgao"`
		Texto    string    `json:"texto"`
		CriadoEm time.Time `json:"criado_em"`
	}

	var lista []IntimacaoItem
	for rows.Next() {
		var item IntimacaoItem
		if err := rows.Scan(&item.ID, &item.Tribunal, &item.Orgao, &item.Texto, &item.CriadoEm); err == nil {
			lista = append(lista, item)
		}
	}

	if lista == nil {
		lista = []IntimacaoItem{}
	}

	JSON(w, http.StatusOK, map[string]any{
		"data":  lista,
		"count": len(lista),
	})
}

func (h *Handler) GetIndices(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	ctx := r.Context()

	sqlQuery := `
SELECT id, sigla, nome, valor, data_referencia
FROM indices_economicos
ORDER BY data_referencia DESC, sigla ASC
LIMIT 20
`
	rows, err := h.Pool.Query(ctx, sqlQuery)
	if err != nil {
		JSON(w, http.StatusOK, map[string]any{"data": []any{}, "count": 0})
		return
	}
	defer rows.Close()

	type IndiceItem struct {
		ID             int64     `json:"id"`
		Sigla          string    `json:"sigla"`
		Nome           string    `json:"nome"`
		Valor          float64   `json:"valor"`
		DataReferencia time.Time `json:"data_referencia"`
	}

	var lista []IndiceItem
	for rows.Next() {
		var item IndiceItem
		if err := rows.Scan(&item.ID, &item.Sigla, &item.Nome, &item.Valor, &item.DataReferencia); err == nil {
			lista = append(lista, item)
		}
	}

	if lista == nil {
		lista = []IndiceItem{}
	}

	JSON(w, http.StatusOK, map[string]any{
		"data":  lista,
		"count": len(lista),
	})
}
