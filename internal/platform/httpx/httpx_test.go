package httpx

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"painelteodoro/internal/platform/logs"
)

func serve(t *testing.T, h http.HandlerFunc, req *http.Request) (*httptest.ResponseRecorder, string) {
	t.Helper()
	var buf bytes.Buffer
	rec := httptest.NewRecorder()
	Chain(h, logs.New(&buf, "info")).ServeHTTP(rec, req)
	return rec, buf.String()
}

func TestCabecalhosDeSeguranca(t *testing.T) {
	rec, _ := serve(t, func(http.ResponseWriter, *http.Request) {}, httptest.NewRequest("GET", "/", nil))
	for _, h := range []string{"Content-Security-Policy", "X-Content-Type-Options", "Referrer-Policy", "Strict-Transport-Security", "X-Request-Id"} {
		if rec.Header().Get(h) == "" {
			t.Errorf("faltou cabeçalho %s", h)
		}
	}
	if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Error("CSP sem frame-ancestors 'none'")
	}
	if strings.Contains(rec.Header().Get("Content-Security-Policy"), "unsafe-inline") {
		t.Error("CSP não pode ter unsafe-inline")
	}
}

func TestLogNaoTemQueryString(t *testing.T) {
	_, out := serve(t, func(http.ResponseWriter, *http.Request) {}, httptest.NewRequest("GET", "/busca?q=fulano+da+silva", nil))
	if strings.Contains(out, "fulano") || strings.Contains(out, "q=") {
		t.Fatalf("log vazou query string: %s", out)
	}
	if !strings.Contains(out, `"caminho":"/busca"`) || !strings.Contains(out, "request_id") {
		t.Fatalf("log sem campos esperados: %s", out)
	}
}

func TestLimiteDeCorpo(t *testing.T) {
	body := strings.NewReader(strings.Repeat("a", maxBody+10))
	var readErr error
	serve(t, func(_ http.ResponseWriter, r *http.Request) { _, readErr = io.ReadAll(r.Body) },
		httptest.NewRequest("POST", "/", body))
	if readErr == nil {
		t.Fatal("corpo acima de 1 MiB deveria falhar")
	}
}
