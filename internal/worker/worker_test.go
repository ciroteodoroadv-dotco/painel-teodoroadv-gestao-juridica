package worker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"painelteodoro/internal/worker"
)

type mockProcessoService struct {
	called bool
	err    error
}

func (m *mockProcessoService) SincronizarProcessos(ctx context.Context) error {
	m.called = true
	return m.err
}

type mockIntimacaoService struct {
	called bool
	err    error
}

func (m *mockIntimacaoService) SincronizarIntimacoes(ctx context.Context) error {
	m.called = true
	return m.err
}

func TestWorkerLifecycle(t *testing.T) {
	procMock := &mockProcessoService{}
	intimMock := &mockIntimacaoService{}

	w := worker.NewWorker(10*time.Millisecond, procMock, intimMock)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("o worker nao respondeu ao cancelamento do contexto a tempo")
	}
}

func TestWorkerRunSync(t *testing.T) {
	procMock := &mockProcessoService{err: errors.New("erro simulado")}
	intimMock := &mockIntimacaoService{}

	w := worker.NewWorker(100*time.Millisecond, procMock, intimMock)
	w.RunSync(context.Background())

	if !procMock.called {
		t.Error("esperava-se que SincronizarProcessos fosse chamado")
	}
	if !intimMock.called {
		t.Error("esperava-se que SincronizarIntimacoes fosse chamado")
	}
}
