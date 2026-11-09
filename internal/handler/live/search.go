package live

import (
	"database/sql"
	"fmt"
	"html"
	"net/http"
	"strings"
)

type SearchHandler struct {
	db *sql.DB
}

func NewSearchHandler(db *sql.DB) *SearchHandler {
	return &SearchHandler{db: db}
}

type ProcessoItem struct {
	ID            int64
	NumeroCNJ     string
	Tribunal      string
	OrgaoJulgador string
	Status        string
}

// BuscarProcessosHTMX responde fragmentos de tabela HTML em tempo real ao digitar na busca
func (h *SearchHandler) BuscarProcessosHTMX(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	sqlQuery := `
SELECT id, numero_cnj, COALESCE(tribunal, ''), COALESCE(orgao_julgador, ''), status
FROM processos
WHERE numero_cnj ILIKE $1 OR tribunal ILIKE $1 OR orgao_julgador ILIKE $1
ORDER BY id DESC LIMIT 20
`

	rows, err := h.db.QueryContext(r.Context(), sqlQuery, "%"+query+"%")
	if err != nil {
		http.Error(w, "Erro na consulta de processos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var lista []ProcessoItem
	for rows.Next() {
		var item ProcessoItem
		if err := rows.Scan(&item.ID, &item.NumeroCNJ, &item.Tribunal, &item.OrgaoJulgador, &item.Status); err == nil {
			lista = append(lista, item)
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if len(lista) == 0 {
		fmt.Fprintf(w, `<tr><td colspan="4" class="text-center py-4 text-gray-500">Nenhum processo encontrado para "%s".</td></tr>`, html.EscapeString(query))
		return
	}

	for _, p := range lista {
		fmt.Fprintf(w, `
<tr class="hover:bg-gray-50 transition-colors border-b">
<td class="py-3 px-4 font-mono font-medium text-blue-900">%s</td>
<td class="py-3 px-4 text-sm text-gray-700">%s</td>
<td class="py-3 px-4 text-sm text-gray-600">%s</td>
<td class="py-3 px-4"><span class="px-2 py-1 text-xs font-semibold rounded bg-blue-100 text-blue-800">%s</span></td>
</tr>
`, p.NumeroCNJ, p.Tribunal, p.OrgaoJulgador, p.Status)
	}
}
