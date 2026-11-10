package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"painelteodoro/internal/repository"
)

type ProcessoLister interface {
	Listar(ctx context.Context, limit, offset int) ([]repository.Processo, error)
}

func ListarProcessos(lister ProcessoLister) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		if lister == nil {
			http.Error(w, "Servico indisponivel", http.StatusServiceUnavailable)
			return
		}

		limit := 10
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		offset := 0
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		lista, err := lister.Listar(r.Context(), limit, offset)
		if err != nil {
			http.Error(w, "Erro ao buscar processos", http.StatusInternalServerError)
			return
		}

		if lista == nil {
			lista = []repository.Processo{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(lista)
	}
}
