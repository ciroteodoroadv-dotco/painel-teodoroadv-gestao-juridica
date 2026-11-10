package handler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"painelteodoro/internal/handler"
)

type mockPinger struct {
	err error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.err
}

func TestReadyCheckOK(t *testing.T) {
	mock := &mockPinger{err: nil}
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlerFunc := handler.ReadyCheck(mock)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusOK)
	}
}

func TestReadyCheckNilDB(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlerFunc := handler.ReadyCheck(nil)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusServiceUnavailable)
	}
}

func TestReadyCheckPingError(t *testing.T) {
	mock := &mockPinger{err: errors.New("db down")}
	req, _ := http.NewRequest(http.MethodGet, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlerFunc := handler.ReadyCheck(mock)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusServiceUnavailable)
	}
}

func TestReadyCheckMethodNotAllowed(t *testing.T) {
	mock := &mockPinger{err: nil}
	req, _ := http.NewRequest(http.MethodPost, "/readyz", nil)
	rr := httptest.NewRecorder()

	handlerFunc := handler.ReadyCheck(mock)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusMethodNotAllowed)
	}
}
