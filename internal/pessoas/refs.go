package pessoas

import (
	"context"

	"painelteodoro/internal/platform/webx"
)

// Refs devolve código e nome de exibição (sem documento) para os seletores de Atendimentos e Tarefas.
// ids nulo = todas as pessoas não excluídas; com ids = exatamente essas (mesmo excluídas, para não sumir o nome
// de quem já está ligado a um registro). Ordem: nome, como as demais listas.
func (s *Service) Refs(ctx context.Context, ids []string) ([]webx.PessoaRef, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, codigo, nome_exibicao FROM pessoas
		WHERE ($2 AND excluido_em IS NULL) OR id = ANY($1::uuid[])
		ORDER BY lower(nome_exibicao), id`, ids, ids == nil)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []webx.PessoaRef
	for rows.Next() {
		var r webx.PessoaRef
		if err := rows.Scan(&r.ID, &r.Codigo, &r.Nome); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
