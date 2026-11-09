package worker_test

import (
	"context"
	"testing"
	"time"

	"painelteodoro/internal/worker"
)

func TestWorkerLifecycle(t *testing.T) {
	w := worker.NewWorker(10 * time.Millisecond)

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
