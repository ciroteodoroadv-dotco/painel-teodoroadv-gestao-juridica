package repository_test

import (
	"context"
	"testing"

	"painelteodoro/internal/repository"
)

func TestProcessoRepository_NilPool(t *testing.T) {
	repo := repository.NewProcessoRepository(nil)
	ctx := context.Background()

	t.Run("Contar com pool nil", func(t *testing.T) {
		_, err := repo.Contar(ctx)
		if err == nil {
			t.Error("esperava erro para pool nil em Contar, obteve nil")
		}
	})

	t.Run("Listar com pool nil", func(t *testing.T) {
		_, err := repo.Listar(ctx, 10, 0)
		if err == nil {
			t.Error("esperava erro para pool nil em Listar, obteve nil")
		}
	})

	t.Run("BuscarPorID com pool nil", func(t *testing.T) {
		_, err := repo.BuscarPorID(ctx, 1)
		if err == nil {
			t.Error("esperava erro para pool nil em BuscarPorID, obteve nil")
		}
	})

	t.Run("Criar com pool nil", func(t *testing.T) {
		p := &repository.Processo{Numero: "123", Cliente: "Teste", Status: "Ativo"}
		err := repo.Criar(ctx, p)
		if err == nil {
			t.Error("esperava erro para pool nil em Criar, obteve nil")
		}
	})

	t.Run("Atualizar com pool nil", func(t *testing.T) {
		p := &repository.Processo{ID: 1, Numero: "123", Cliente: "Teste", Status: "Ativo"}
		err := repo.Atualizar(ctx, p)
		if err == nil {
			t.Error("esperava erro para pool nil em Atualizar, obteve nil")
		}
	})
}

func TestErrProcessoNaoEncontrado(t *testing.T) {
	if repository.ErrProcessoNaoEncontrado == nil {
		t.Error("ErrProcessoNaoEncontrado nao deve ser nil")
	}
	if repository.ErrProcessoNaoEncontrado.Error() != "processo nao encontrado" {
		t.Errorf("mensagem de erro inesperada: %s", repository.ErrProcessoNaoEncontrado.Error())
	}
}
