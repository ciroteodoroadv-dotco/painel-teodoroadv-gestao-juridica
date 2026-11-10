package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"painelteodoro/internal/handler"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/healthz", nil)
	if err != nil {
		t.Fatalf("erro ao criar requisicao: %v", err)
	}

	rr := httptest.NewRecorder()
	handlerFunc := http.HandlerFunc(handler.HealthCheck)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Content-Type incorreto: recebido %v, esperado application/json", contentType)
	}
}

func TestHealthCheckMethodNotAllowed(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/healthz", nil)
	if err != nil {
		t.Fatalf("erro ao criar requisicao: %v", err)
	}

	rr := httptest.NewRecorder()
	handlerFunc := http.HandlerFunc(handler.HealthCheck)
	handlerFunc.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("codigo de status incorreto: recebido %v, esperado %v", status, http.StatusMethodNotAllowed)
	}
}
