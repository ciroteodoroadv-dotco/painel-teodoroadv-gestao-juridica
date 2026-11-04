package atendimentos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/codigos"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de atendimentos. Só toca nas próprias tabelas.
type Service struct {
	Pool    *pgxpool.Pool
	Pessoas PessoaLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "atendimento",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// conferePessoas exige que todas as pessoas citadas existam e não estejam excluídas.
func (s *Service) conferePessoas(ctx context.Context, n normalized) error {
	quer := append(append(append([]string{}, n.clientes...), n.partes...), n.criadoPor)
	if n.criadoPor == "" {
		quer = quer[:len(quer)-1]
	}
	for _, id := range quer {
		if !uuidValido(id) {
			return webx.Invalidf("Pessoa não encontrada.")
		}
	}
	refs, err := s.Pessoas.Refs(ctx, quer)
	if err != nil {
		return err
	}
	achou := map[string]bool{}
	for _, r := range refs {
		achou[r.ID] = true
	}
	for _, id := range quer {
		if !achou[id] {
			return webx.Invalidf("Pessoa não encontrada.")
		}
	}
	return nil
}

func uuidValido(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func gravarPessoas(ctx context.Context, tx pgx.Tx, id string, n normalized) error {
	if _, err := tx.Exec(ctx, `DELETE FROM atendimento_pessoas WHERE atendimento_id=$1`, id); err != nil {
		return err
	}
	for papel, ids := range map[string][]string{"cliente": n.clientes, "parte_adversa": n.partes} {
		for i, p := range ids {
			if _, err := tx.Exec(ctx, `INSERT INTO atendimento_pessoas (atendimento_id, pessoa_id, papel, ordem) VALUES ($1,$2,$3,$4)`,
				id, p, papel, i); err != nil {
				return err
			}
		}
	}
	return nil
}

// Create cadastra o atendimento (tipo e status padrão) e devolve o id e o código (ATE. 0000001).
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (id, codigo string, err error) {
	n, err := in.normalize()
	if err != nil {
		return "", "", err
	}
	if err := s.conferePessoas(ctx, n); err != nil {
		return "", "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if codigo, err = codigos.Proximo(ctx, tx, "ATE"); err != nil {
		return "", "", err
	}
	err = tx.QueryRow(ctx, `INSERT INTO atendimentos (codigo, tipo, status, descricao, observacoes, data_abertura,
		criado_por_pessoa_id, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8) RETURNING id`,
		codigo, TipoPadrao, StatusPadrao, n.descricao, n.obs, n.data, webx.Nullable(n.criadoPor), ac.UserID).Scan(&id)
	if err == nil {
		err = gravarPessoas(ctx, tx, id, n)
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "atendimento.criado", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		return "", "", err
	}
	return id, codigo, nil
}

// Update altera com concorrência otimista; as listas de clientes e partes adversas são substituídas.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	n, err := in.normalize()
	if err != nil {
		return err
	}
	if err := s.conferePessoas(ctx, n); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE atendimentos SET descricao=$3, observacoes=$4, data_abertura=$5, criado_por_pessoa_id=$6,
		updated_at=now(), updated_by=$7, version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.descricao, n.obs, n.data, webx.Nullable(n.criadoPor), ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var one int
		e := s.Pool.QueryRow(ctx, `SELECT 1 FROM atendimentos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
		if errors.Is(e, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		if e != nil {
			return e
		}
		return webx.ErrConflict
	}
	if err = gravarPessoas(ctx, tx, id, n); err == nil {
		err = s.record(ctx, tx, ac, "atendimento.atualizado", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return err
}
