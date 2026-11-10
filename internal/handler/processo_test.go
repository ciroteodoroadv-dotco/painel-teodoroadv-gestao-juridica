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

type mockFinder struct {
	processo *repository.Processo
	err      error
}

func (m *mockFinder) BuscarPorID(ctx context.Context, id int64) (*repository.Processo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.processo, nil
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

func TestBuscarProcessoPorID_Sucesso(t *testing.T) {
	mock := &mockFinder{
		processo: &repository.Processo{ID: 1, Numero: "0000001-00.2026.8.26.0344", Cliente: "Cliente A", Status: "Ativo", CreatedAt: time.Now()},
	}

	h := handler.BuscarProcessoPorID(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/processos/detalhe?id=1", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("esperava status 200, obteve %d", rec.Code)
	}
}

func TestBuscarProcessoPorID_MetodoInvalido(t *testing.T) {
	h := handler.BuscarProcessoPorID(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/processos/detalhe", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("esperava status 405, obteve %d", rec.Code)
	}
}

func TestBuscarProcessoPorID_FinderNil(t *testing.T) {
	h := handler.BuscarProcessoPorID(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/processos/detalhe?id=1", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("esperava status 503, obteve %d", rec.Code)
	}
}

func TestBuscarProcessoPorID_IDInvalido(t *testing.T) {
	mock := &mockFinder{}
	h := handler.BuscarProcessoPorID(mock)

	testes := []string{"", "abc", "-5", "0"}
	for _, idStr := range testes {
		url := "/api/processos/detalhe"
		if idStr != "" {
			url += "?id=" + idStr
		}
		req := httptest.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("para id '%s' esperava status 400, obteve %d", idStr, rec.Code)
		}
	}
}

func TestBuscarProcessoPorID_NaoEncontrado(t *testing.T) {
	mock := &mockFinder{err: repository.ErrProcessoNaoEncontrado}
	h := handler.BuscarProcessoPorID(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/processos/detalhe?id=999", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("esperava status 404, obteve %d", rec.Code)
	}
}

func TestBuscarProcessoPorID_ErroInterno(t *testing.T) {
	mock := &mockFinder{err: errors.New("falha no banco")}
	h := handler.BuscarProcessoPorID(mock)
	req := httptest.NewRequest(http.MethodGet, "/api/processos/detalhe?id=1", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("esperava status 500, obteve %d", rec.Code)
	}
}
