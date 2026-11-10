package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"painelteodoro/internal/handler"
	"painelteodoro/internal/repository"
)

type mockLister struct {
	processos []repository.Processo
	err       error
}

func (m *mockLister) Listar(ctx context.Context, limit, offset int) ([]repository.Processo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.processos, nil
}

func TestListarProcessos_Sucesso(t *testing.T) {
	mock := &mockLister{
		processos: []repository.Processo{
			{ID: 1, Numero: "0000001-00.2026.8.26.0344", Cliente: "Cliente A", Status: "Ativo", CreatedAt: time.Now()},
		},
	}

	h := handler.ListarProcessos(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/processos?limit=5&offset=0", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperava status 200, obteve %d", rec.Code)
	}
}

func TestListarProcessos_MetodoInvalido(t *testing.T) {
	h := handler.ListarProcessos(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/processos", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("esperava status 405, obteve %d", rec.Code)
	}
}

func TestListarProcessos_ListerNil(t *testing.T) {
	h := handler.ListarProcessos(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/processos", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("esperava status 503, obteve %d", rec.Code)
	}
}

func TestListarProcessos_ErroRepository(t *testing.T) {
	mock := &mockLister{err: errors.New("erro no banco")}
	h := handler.ListarProcessos(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/processos", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("esperava status 500, obteve %d", rec.Code)
	}
}
