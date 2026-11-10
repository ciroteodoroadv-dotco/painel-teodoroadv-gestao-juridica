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

func TestProcessoRepository_Listar_NilDB(t *testing.T) {
	repo := repository.NewProcessoRepository(nil)

	_, err := repo.Listar(context.Background(), 10, 0)
	if err == nil {
		t.Error("esperava erro ao chamar Listar com pool nil, mas obteve nil")
	}
}
