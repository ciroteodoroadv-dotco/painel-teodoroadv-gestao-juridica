package health

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDB struct{ err error }

func (f fakeDB) Ping(context.Context) error { return f.err }

func get(t *testing.T, db Pinger, path string) int {
	t.Helper()
	mux := http.NewServeMux()
	Register(mux, db, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code
}

func TestHealthzSempreOK(t *testing.T) {
	if c := get(t, fakeDB{err: errors.New("x")}, "/healthz"); c != 200 {
		t.Fatalf("healthz = %d", c)
	}
}

func TestReadyz(t *testing.T) {
	if c := get(t, fakeDB{}, "/readyz"); c != 200 {
		t.Fatalf("readyz com banco ok = %d", c)
	}
	if c := get(t, fakeDB{err: errors.New("caiu")}, "/readyz"); c != 503 {
		t.Fatalf("readyz com banco fora = %d", c)
	}
}
