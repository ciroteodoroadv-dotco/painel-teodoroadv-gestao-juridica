package processos

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// PorNumero devolve o id do processo não excluído cujo número CNJ (já normalizado) é numero, ou "" se não houver.
// Audiências e diligências guardam o número digitado e, existindo o processo, também o vínculo real.
func (s *Service) PorNumero(ctx context.Context, numero string) (string, error) {
	if strings.TrimSpace(numero) == "" {
		return "", nil
	}
	var id string
	err := s.Pool.QueryRow(ctx, `SELECT id::text FROM processos WHERE excluido_em IS NULL AND numero_cnj=$1 ORDER BY created_at LIMIT 1`, numero).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}

// Rotulo devolve o texto curto do processo (número, senão título, senão código); ok=false se não existir.
func (s *Service) Rotulo(ctx context.Context, id string) (string, bool, error) {
	m, err := s.Resumos(ctx, []string{id})
	if err != nil {
		return "", false, err
	}
	r, ok := m[id]
	return r, ok, nil
}

// OpcoesCompromisso monta o seletor "Processo vinculado" do Novo compromisso: "PRO. 0000001 · número · título" (B).
func (s *Service) OpcoesCompromisso(ctx context.Context) ([]Opcao, error) {
	ps, err := s.Listar(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Opcao, len(ps))
	for i, p := range ps {
		var partes []string
		for _, x := range []string{p.Codigo, p.Numero, p.Titulo} {
			if x != "" {
				partes = append(partes, x)
			}
		}
		out[i] = Opcao{Valor: p.ID, Rotulo: strings.Join(partes, " · ")}
	}
	return out, nil
}
