package pessoas

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"painelteodoro/internal/audit"
	"painelteodoro/internal/platform/br"
	"painelteodoro/internal/platform/crypt"
	"painelteodoro/internal/platform/webx"
)

const purpose = "pessoas.documento"

// ProcessoRef é uma referência a processo vinda de OUTRO domínio (interface, nunca SQL cruzado).
type ProcessoRef struct{ ID, Numero, Qualidade string }

// ProcessosLookup é implementado pelo domínio de processos.
type ProcessosLookup interface {
	ProcessosDaPessoa(ctx context.Context, pessoaID string) ([]ProcessoRef, error)
}

// Service concentra as regras do cadastro.
type Service struct {
	Pool      *pgxpool.Pool
	FieldKey  []byte
	Processos ProcessosLookup // opcional (nil em testes isolados)
}

func (s *Service) seal(doc string) (enc, idx []byte, err error) {
	if doc == "" {
		return nil, nil, nil
	}
	if enc, err = crypt.Seal(crypt.Derive(s.FieldKey, purpose), []byte(doc)); err != nil {
		return nil, nil, err
	}
	return enc, crypt.BlindIndex(s.FieldKey, purpose, doc), nil
}

func (s *Service) unseal(enc []byte) string {
	if len(enc) == 0 {
		return ""
	}
	b, err := crypt.Open(crypt.Derive(s.FieldKey, purpose), enc)
	if err != nil {
		return ""
	}
	return string(b)
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, ac webx.Actor, action, id string) error {
	return audit.Record(ctx, tx, audit.Event{Actor: ac.UserID, Action: action, TargetType: "pessoa",
		TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}

// Create cadastra uma pessoa; a escrita e a auditoria acontecem na mesma transação.
func (s *Service) Create(ctx context.Context, ac webx.Actor, in Input) (string, error) {
	n, err := in.normalize()
	if err != nil {
		return "", err
	}
	enc, idx, err := s.seal(n.doc)
	if err != nil {
		return "", err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO pessoas (tipo, nome, documento_enc, documento_idx, email, telefone, papel,
		retencao_ate, created_by, updated_by) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$9) RETURNING id`,
		n.tipo, n.nome, enc, idx, webx.Nullable(n.email), webx.Nullable(n.tel), n.papel, n.retencao, ac.UserID).Scan(&id)
	if webx.IsUniqueViolation(err) {
		return "", webx.Invalidf("Já existe uma pessoa cadastrada com este documento.")
	}
	if err == nil {
		err = s.record(ctx, tx, ac, "pessoa.criada", id)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	return id, err
}

// Update altera com concorrência otimista: se a versão mudou, devolve webx.ErrConflict.
func (s *Service) Update(ctx context.Context, ac webx.Actor, id string, version int, in Input) error {
	n, err := in.normalize()
	if err != nil {
		return err
	}
	enc, idx, err := s.seal(n.doc)
	if err != nil {
		return err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE pessoas SET tipo=$3, nome=$4, documento_enc=$5, documento_idx=$6, email=$7,
		telefone=$8, papel=$9, retencao_ate=$10, updated_at=now(), updated_by=$11, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`,
		id, version, n.tipo, n.nome, enc, idx, webx.Nullable(n.email), webx.Nullable(n.tel), n.papel, n.retencao, ac.UserID)
	if webx.IsUniqueViolation(err) {
		return webx.Invalidf("Já existe uma pessoa cadastrada com este documento.")
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "pessoa.alterada", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Service) whyNoRows(ctx context.Context, id string) error {
	var one int
	err := s.Pool.QueryRow(ctx, `SELECT 1 FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return webx.ErrNotFound
	}
	if err != nil {
		return err
	}
	return webx.ErrConflict
}

// Delete marca a pessoa como excluída (sem apagar). Recusa se ainda for parte de processo.
func (s *Service) Delete(ctx context.Context, ac webx.Actor, id string, version int) error {
	if s.Processos != nil {
		refs, err := s.Processos.ProcessosDaPessoa(ctx, id)
		if err != nil {
			return err
		}
		if len(refs) > 0 {
			return webx.Invalidf("Esta pessoa é parte em processo(s). Retire-a dos processos antes de excluir.")
		}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `UPDATE pessoas SET excluido_em=now(), updated_at=now(), updated_by=$3, version=version+1
		WHERE id=$1 AND version=$2 AND excluido_em IS NULL`, id, version, ac.UserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.whyNoRows(ctx, id)
	}
	if err = s.record(ctx, tx, ac, "pessoa.excluida", id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Reveal devolve o documento em claro e registra o acesso na auditoria (LGPD: acesso a dado sensível).
func (s *Service) Reveal(ctx context.Context, ac webx.Actor, id string) (string, error) {
	var enc []byte
	err := s.Pool.QueryRow(ctx, `SELECT documento_enc FROM pessoas WHERE id=$1 AND excluido_em IS NULL`, id).Scan(&enc)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", webx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	doc := s.unseal(enc)
	if doc == "" {
		return "", nil
	}
	return br.Digits(doc), audit.RecordPool(ctx, s.Pool, audit.Event{Actor: ac.UserID, Action: "pessoa.documento_revelado",
		TargetType: "pessoa", TargetID: id, Result: audit.OK, RequestID: ac.RequestID})
}
