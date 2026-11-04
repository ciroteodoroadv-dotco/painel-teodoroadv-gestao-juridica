package intimacoes

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"painelteodoro/internal/auth"
	"painelteodoro/internal/platform/webx"
)

// AvisoDataJud é dito no detalhe de toda movimentação do DataJud: a fonte não traz o teor da intimação.
const AvisoDataJud = "Movimentação processual (DataJud). Esta fonte não fornece o teor da intimação."

// Detalhe é o drawer "Intimação": Origem, Tribunal, Órgão, Processo, Data, Tipo, Texto completo e Link.
type Detalhe struct {
	Origem, Tribunal, Orgao, Processo, Data, Tipo, Texto, Link, AvisoFonte string
}

func traco(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// Detalhe lê a intimação. Sem texto integral (registro anterior à migração 0013), usa o resumo.
func (s *Service) Detalhe(ctx context.Context, id string) (Detalhe, error) {
	if !webx.UUID(id) {
		return Detalhe{}, webx.ErrNotFound
	}
	var d Detalhe
	var data *time.Time
	err := s.Pool.QueryRow(ctx, `SELECT origem, tribunal, orgao, processo_numero, data_publicacao, tipo, coalesce(nullif(texto,''), resumo), link
		FROM intimacoes WHERE id=$1::uuid`, id).Scan(&d.Origem, &d.Tribunal, &d.Orgao, &d.Processo, &data, &d.Tipo, &d.Texto, &d.Link)
	if errors.Is(err, pgx.ErrNoRows) {
		return Detalhe{}, webx.ErrNotFound
	}
	if err != nil {
		return Detalhe{}, err
	}
	d.Data = "—"
	if data != nil {
		d.Data = webx.DataBR(*data)
	}
	if d.Origem == "DataJud" {
		d.AvisoFonte = AvisoDataJud
	}
	d.Tribunal, d.Orgao, d.Processo, d.Tipo, d.Texto = traco(d.Tribunal), traco(d.Orgao), traco(d.Processo), traco(d.Tipo), traco(d.Texto)
	return d, nil
}

func (h *Handler) detalhe(w http.ResponseWriter, r *http.Request, _ *auth.Session) {
	d, err := h.S.Detalhe(r.Context(), r.PathValue("id"))
	if webx.Falha(w, err, "Intimação não encontrada.", "Falha ao abrir a intimação.") {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := fragmento("detalhe.html").ExecuteTemplate(w, "detalhe", d); err != nil {
		http.Error(w, "Erro interno.", http.StatusInternalServerError)
	}
}
