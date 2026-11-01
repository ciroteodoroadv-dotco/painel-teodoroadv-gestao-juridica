package processos

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/pessoas"
	"painelteodoro/internal/platform/webx"
)

// Service concentra as regras de processos. Só toca nas próprias tabelas.
type Service struct {
	Pool    *pgxpool.Pool
	Pessoas PessoaLookup
	Users   UserLookup
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string, meta map[string]string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "processo",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID, Meta: meta})
}

func (s *Service) checkResponsavel(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	m, err := s.Users.Emails(ctx, []string{id})
	if err != nil {
		return err
	}
	if _, ok := m[id]; !ok {
		return webx.Invalidf("Responsável inválido.")
	}
	return nil
}

// Create cadastra o processo (partes entram depois, uma a uma, com auditoria própria).
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	if err := s.checkResponsavel(ctx, n.responsavel); err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO processos (numero_cnj, orgao, situacao, responsavel_id, created_by, updated_by)
		VALUES ($1,$2,$3,$4,$5,$5) RETURNING id`, n.numero, n.orgao, n.situacao, webx.Nullable(n.responsavel), ac.UserID).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", webx.Invalidf("Já existe um processo cadastrado com este número.")
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "processo.criado", id, nil)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return id, err
}

// Update altera com concorrência otimista.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	n, err := in.normalize()
	if err != nil {
		return err
	}
	if err := s.checkResponsavel(ctx, n.responsavel); err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE processos SET numero_cnj=$3, orgao=$4, situacao=$5, responsavel_id=$6,
		updated_at=now(), updated_by=$7, version=version+1 WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.numero, n.orgao, n.situacao, webx.Nullable(n.responsavel), ac.UserID)
	if webx.IsUniqueViolation(err) {
		return webx.Invalidf("Já existe um processo cadastrado com este número.")
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "processo.alterado", id, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) whyNoRows(ctx context.Context, id string) error {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM processos WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return webx.ErrConflict
}

// Delete marca como excluído (sem apagar).
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE processos SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "processo.excluido", id, nil); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// AddParte liga uma pessoa ao processo. Duas escritas (parte + auditoria) na MESMA transação.
func (s *Service) AddParte(ctx context.Context, ac webx.Actor, processoID, pessoaID, qualidade string) error {
	if rotulo(Qualidades, qualidade) == qualidade {
		return webx.Invalidf("Escolha a qualidade da parte.")
	}
	ok, err := s.Pessoas.Exist(ctx, []string{pessoaID})
	if err != nil {
		return err
	}
	if !ok[pessoaID] {
		return webx.Invalidf("Pessoa não encontrada.")
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var one int
	if err := tx.QueryRow(ctx, `SELECT 1 FROM processos WHERE id=$1 AND excluido_em IS NULL FOR UPDATE`, processoID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return webx.ErrNotFound
		}
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO processo_partes (processo_id, pessoa_id, qualidade) VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING`, processoID, pessoaID, qualidade); err != nil {
		return err
	}
	if err = s.record(ctx, tx, ac, "processo.parte_adicionada", processoID, map[string]string{"papel": qualidade}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// RemoveParte desliga a pessoa do processo naquela qualidade.
func (s *Service) RemoveParte(ctx context.Context, ac webx.Actor, processoID, pessoaID, qualidade string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `DELETE FROM processo_partes WHERE processo_id=$1 AND pessoa_id=$2 AND qualidade=$3`,
		processoID, pessoaID, qualidade)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return webx.ErrNotFound
	}
	if err = s.record(ctx, tx, ac, "processo.parte_removida", processoID, map[string]string{"papel": qualidade}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ pessoas.ProcessosLookup = (*Service)(nil) // este domínio atende o contrato que pessoas exige
