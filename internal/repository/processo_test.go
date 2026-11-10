package repository_test

import (
	"context"
	"testing"

	"painelteodoro/internal/repository"
)

func TestProcessoRepository_Contar_NilDB(t *testing.T) {
	repo := repository.NewProcessoRepository(nil)

	_, err := repo.Contar(context.Background())
	if err == nil {
		t.Error("esperava erro ao chamar Contar com pool nil, mas obteve nil")
	}
}
