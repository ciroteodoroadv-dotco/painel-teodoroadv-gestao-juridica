package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"painelteodoro/internal/repository"
)

type ProcessoLister interface {
	Listar(ctx context.Context, limit, offset int) ([]repository.Processo, error)
}

type ProcessoFinder interface {
	BuscarPorID(ctx context.Context, id int64) (*repository.Processo, error)
}

type ProcessoCreator interface {
	Criar(ctx context.Context, p *repository.Processo) error
}

type ProcessoUpdater interface {
	Atualizar(ctx context.Context, p *repository.Processo) error
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

func BuscarProcessoPorID(finder ProcessoFinder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		if finder == nil {
			http.Error(w, "Servico indisponivel", http.StatusServiceUnavailable)
			return
		}

		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "ID nao informado", http.StatusBadRequest)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "ID invalido", http.StatusBadRequest)
			return
		}

		processo, err := finder.BuscarPorID(r.Context(), id)
		if err != nil {
			if errors.Is(err, repository.ErrProcessoNaoEncontrado) {
				http.Error(w, "Processo nao encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, "Erro ao buscar processo", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(processo)
	}
}

func CriarProcesso(creator ProcessoCreator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		if creator == nil {
			http.Error(w, "Servico indisponivel", http.StatusServiceUnavailable)
			return
		}

		var p repository.Processo
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Payload JSON invalido", http.StatusBadRequest)
			return
		}

		if err := creator.Criar(r.Context(), &p); err != nil {
			http.Error(w, "Erro ao criar processo", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(p)
	}
}

func AtualizarProcesso(updater ProcessoUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut && r.Method != http.MethodPatch {
			http.Error(w, "Metodo nao permitido", http.StatusMethodNotAllowed)
			return
		}

		if updater == nil {
			http.Error(w, "Servico indisponivel", http.StatusServiceUnavailable)
			return
		}

		var p repository.Processo
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Payload JSON invalido", http.StatusBadRequest)
			return
		}

		if p.ID <= 0 {
			http.Error(w, "ID invalido", http.StatusBadRequest)
			return
		}

		if err := updater.Atualizar(r.Context(), &p); err != nil {
			if errors.Is(err, repository.ErrProcessoNaoEncontrado) {
				http.Error(w, "Processo nao encontrado", http.StatusNotFound)
				return
			}
			http.Error(w, "Erro ao atualizar processo", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(p)
	}
}
