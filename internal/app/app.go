// Package app monta os domínios (pessoas, processos, tarefas) sobre a autenticação.
// É o único lugar que conhece todos: os domínios só se falam por interfaces.
package app

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/agenda"
	"painelteodoro/internal/atendimentos"
	"painelteodoro/internal/auth"
	"painelteodoro/internal/documentos"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/webx"
	"painelteodoro/internal/prazos"
	"painelteodoro/internal/processos"
	"painelteodoro/internal/tarefas"
)

// userLookup adapta o cadastro de usuários às interfaces que os domínios exigem.
type userLookup struct{ s *auth.Store }

func (u userLookup) Options(ctx context.Context) ([]webx.Opcao, error) {
	list, err := u.s.UserOptions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]webx.Opcao, len(list))
	for i, o := range list {
		out[i] = webx.Opcao{Valor: o.ID, Rotulo: o.Email}
	}
	return out, nil
}

func (u userLookup) Emails(ctx context.Context, ids []string) (map[string]string, error) {
	return u.s.UserEmails(ctx, ids)
}

// RegisterDomains liga todos os domínios ao roteador. fieldKey é a chave mestra de cifra de campos.
func RegisterDomains(a *auth.Auth, rt *auth.Router, pool *pgxpool.Pool, fieldKey []byte, docsDir string) error {
	users := userLookup{s: a.Store}
	pes := &pessoas.Service{Pool: pool, FieldKey: fieldKey}
	proc := &processos.Service{Pool: pool, Pessoas: pes}
	pes.Papeis = proc // os papéis da pessoa derivam dos processos (decisão do painel B)
	tar := &tarefas.Service{Pool: pool, Processos: proc, Users: users}
	praz := &prazos.Service{Pool: pool, Processos: proc, Users: users}
	ate := &atendimentos.Service{Pool: pool, Pessoas: pes}
	proc.Atendimentos = ate
	proc.Vinculos = []processos.VinculosLookup{tar, praz}

	ag := &agenda.Service{Pool: pool, Processos: proc, Pessoas: pes, Users: users}
	store, err := documentos.NewStore(docsDir, fieldKey)
	if err != nil {
		return err
	}
	doc := &documentos.Service{Pool: pool, Store: store, Processos: proc, Pessoas: pes}

	(&prazos.Handler{A: a, S: praz}).Register(rt) // 1º: widget no topo de "Hoje"
	(&pessoas.Handler{A: a, S: pes}).Register(rt)
	(&atendimentos.Handler{A: a, S: ate}).Register(rt)
	(&processos.Handler{A: a, S: proc}).Register(rt)
	(&tarefas.Handler{A: a, S: tar}).Register(rt)
	(&agenda.Handler{A: a, S: ag}).Register(rt)
	(&documentos.Handler{A: a, S: doc}).Register(rt)
	return nil
}
