package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"painelteodoro/internal/api"
)

func TestJSONResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	api.JSON(rec, http.StatusOK, map[string]string{"status": "ok"})

	if rec.Code != http.StatusOK {
		t.Errorf("esperava status 200, obteve %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type incorreto: %s", ct)
	}
}

func TestErrorResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	api.Error(rec, http.StatusBadRequest, "requisicao invalida")

	if rec.Code != http.StatusBadRequest {
		t.Errorf("esperava status 400, obteve %d", rec.Code)
	}
}
